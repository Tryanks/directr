package flow

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func LoadDefinitionDSL(path string) (Definition, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Definition{}, err
	}
	return ParseDefinitionDSL(raw)
}

func ParseDefinitionDSL(raw []byte) (Definition, error) {
	parser := dslParser{
		definition: Definition{States: map[string]State{}},
	}
	return parser.parse(raw)
}

type dslParser struct {
	definition Definition
	states     map[string]*State
	stack      []dslScope

	lineNo            int
	currentState      string
	currentTransition *Transition
}

type dslScope string

const (
	scopeFlow       dslScope = "flow"
	scopeState      dslScope = "state"
	scopeScene      dslScope = "scene"
	scopeTransition dslScope = "transition"
)

func (p *dslParser) parse(raw []byte) (Definition, error) {
	p.states = map[string]*State{}
	lines := strings.Split(string(raw), "\n")
	for i := range lines {
		p.lineNo = i + 1
		if err := p.parseLine(lines[i]); err != nil {
			return Definition{}, err
		}
	}
	if len(p.stack) != 0 {
		return Definition{}, fmt.Errorf("line %d: unmatched block close, missing '}'", p.lineNo)
	}

	def := p.definition
	def.States = make(map[string]State, len(p.states))
	for name, state := range p.states {
		def.States[name] = *state
	}
	def.Normalize()
	if err := def.Validate(); err != nil {
		return Definition{}, err
	}
	return def, nil
}

func (p *dslParser) parseLine(line string) error {
	line = stripScriptComment(line)
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	open := false
	if strings.HasSuffix(line, "{") {
		open = true
		line = strings.TrimSpace(strings.TrimSuffix(line, "{"))
		if line == "" {
			return nil
		}
	}

	if line == "}" {
		return p.closeScope()
	}
	if open {
		return p.openScope(line)
	}

	return p.parseStatement(line)
}

func (p *dslParser) openScope(header string) error {
	tokens := splitScriptTokens(header)
	if len(tokens) == 0 {
		return fmt.Errorf("line %d: empty block header", p.lineNo)
	}
	keyword := strings.ToLower(strings.TrimSuffix(tokens[0], ":"))

	scope := p.currentScope()
	switch keyword {
	case "flow":
		if scope != "" {
			return fmt.Errorf("line %d: nested flow is not allowed", p.lineNo)
		}
		if len(tokens) < 2 {
			return fmt.Errorf("line %d: flow requires a name", p.lineNo)
		}
		p.definition.Name = unquoteScriptToken(tokens[1])
		p.stack = append(p.stack, scopeFlow)
	case "state":
		if scope != scopeFlow {
			return fmt.Errorf("line %d: state must be defined in flow scope", p.lineNo)
		}
		if len(tokens) < 2 {
			return fmt.Errorf("line %d: state requires a name", p.lineNo)
		}
		name := unquoteScriptToken(tokens[1])
		if name == "" {
			return fmt.Errorf("line %d: state name cannot be empty", p.lineNo)
		}
		if _, ok := p.states[name]; ok {
			return fmt.Errorf("line %d: duplicate state %q", p.lineNo, name)
		}
		p.currentState = name
		p.states[name] = &State{Name: name}
		p.stack = append(p.stack, scopeState)
	case "scene":
		if scope != scopeState {
			return fmt.Errorf("line %d: scene must be inside state", p.lineNo)
		}
		p.stack = append(p.stack, scopeScene)
	case "transition":
		if scope != scopeState {
			return fmt.Errorf("line %d: transition must be inside state", p.lineNo)
		}
		if p.currentState == "" {
			return fmt.Errorf("line %d: transition has no containing state", p.lineNo)
		}
		to, err := p.parseTransitionTarget(tokens)
		if err != nil {
			return err
		}
		p.currentTransition = &Transition{To: to}
		p.stack = append(p.stack, scopeTransition)
	default:
		return fmt.Errorf("line %d: invalid block header %q", p.lineNo, tokens[0])
	}
	return nil
}

func (p *dslParser) parseStatement(line string) error {
	tokens := splitScriptTokens(line)
	if len(tokens) == 0 {
		return nil
	}

	scope := p.currentScope()
	switch scope {
	case scopeFlow:
		return p.parseFlowStatement(tokens)
	case scopeState:
		return p.parseStateStatement(tokens)
	case scopeScene:
		return p.parseSceneStatement(tokens)
	case scopeTransition:
		return p.parseTransitionStatement(tokens)
	default:
		return fmt.Errorf("line %d: statement outside of flow", p.lineNo)
	}
}

func (p *dslParser) parseFlowStatement(tokens []string) error {
	if len(tokens) == 0 {
		return nil
	}
	key := strings.ToLower(strings.TrimSuffix(tokens[0], ":"))
	args := tokens[1:]

	switch key {
	case "initial":
		if len(args) == 0 {
			return fmt.Errorf("line %d: initial requires value", p.lineNo)
		}
		p.definition.Initial = unquoteScriptToken(args[0])
	case "target":
		if len(args) == 0 {
			return fmt.Errorf("line %d: target requires value", p.lineNo)
		}
		p.definition.Target = unquoteScriptToken(args[0])
	case "maxsteps":
		if len(args) == 0 {
			return fmt.Errorf("line %d: maxSteps requires value", p.lineNo)
		}
		maxSteps, err := strconv.Atoi(unquoteScriptToken(args[0]))
		if err != nil {
			return fmt.Errorf("line %d: invalid maxSteps %q", p.lineNo, args[0])
		}
		p.definition.MaxSteps = maxSteps
	case "state":
		return fmt.Errorf("line %d: state block must use opening brace", p.lineNo)
	default:
		return fmt.Errorf("line %d: unsupported flow statement %q", p.lineNo, key)
	}
	return nil
}

func (p *dslParser) parseStateStatement(tokens []string) error {
	if len(tokens) == 0 {
		return nil
	}
	key := strings.ToLower(tokens[0])
	if key == "scene" {
		return fmt.Errorf("line %d: scene must use opening brace", p.lineNo)
	}
	if key == "transition" {
		return fmt.Errorf("line %d: transition must use opening brace", p.lineNo)
	}
	return fmt.Errorf("line %d: unsupported state statement %q", p.lineNo, tokens[0])
}

func (p *dslParser) parseSceneStatement(tokens []string) error {
	state, ok := p.states[p.currentState]
	if !ok {
		return fmt.Errorf("line %d: missing state for scene", p.lineNo)
	}
	if len(tokens) == 0 {
		return nil
	}
	key := strings.ToLower(strings.TrimSuffix(tokens[0], ":"))
	body := tokens[1:]

	switch key {
	case "required":
		if len(body) == 0 {
			return fmt.Errorf("line %d: required must include selector fields", p.lineNo)
		}
		selector := &Selector{}
		if err := parseSelectorAssignments(body, selector, p.lineNo); err != nil {
			return err
		}
		state.Scene.Required = append(state.Scene.Required, *selector)
	case "forbidden":
		if len(body) == 0 {
			return fmt.Errorf("line %d: forbidden must include selector fields", p.lineNo)
		}
		selector := &Selector{}
		if err := parseSelectorAssignments(body, selector, p.lineNo); err != nil {
			return err
		}
		state.Scene.Forbidden = append(state.Scene.Forbidden, *selector)
	case "optional":
		if len(body) == 0 {
			return fmt.Errorf("line %d: optional must include selector fields", p.lineNo)
		}
		selector := &Selector{}
		weight := 1.0
		if err := parseOptionalSelectorAssignments(body, selector, &weight, p.lineNo); err != nil {
			return err
		}
		state.Scene.Optional = append(state.Scene.Optional, WeightedSelector{
			Selector: *selector,
			Weight:   weight,
		})
	default:
		return fmt.Errorf("line %d: unsupported scene statement %q", p.lineNo, key)
	}
	return nil
}

func (p *dslParser) parseTransitionStatement(tokens []string) error {
	if len(tokens) == 0 {
		return nil
	}
	if p.currentTransition == nil {
		return fmt.Errorf("line %d: transition not initialized", p.lineNo)
	}
	key := strings.ToLower(strings.TrimSuffix(tokens[0], ":"))
	args := tokens[1:]
	if key == "wait" {
		if len(args) == 0 || args[0] == "" {
			return fmt.Errorf("line %d: wait requires milliseconds", p.lineNo)
		}
		waitMS, err := parseDurationMS(args[0], p.lineNo)
		if err != nil {
			return err
		}
		p.currentTransition.WaitMS = waitMS
		return nil
	}
	if key == "action" {
		if len(args) < 1 {
			return fmt.Errorf("line %d: action missing type", p.lineNo)
		}
		key = strings.ToLower(args[0])
		args = args[1:]
	}
	action, err := parseAction(key, args, p.lineNo)
	if err != nil {
		return err
	}
	p.currentTransition.Action = action
	return nil
}

func (p *dslParser) closeScope() error {
	if len(p.stack) == 0 {
		return fmt.Errorf("line %d: unmatched '}'", p.lineNo)
	}
	n := len(p.stack) - 1
	scope := p.stack[n]
	p.stack = p.stack[:n]

	switch scope {
	case scopeTransition:
		if p.currentState == "" {
			return fmt.Errorf("line %d: transition has no state", p.lineNo)
		}
		state, ok := p.states[p.currentState]
		if !ok || p.currentTransition == nil {
			return fmt.Errorf("line %d: transition close without start", p.lineNo)
		}
		state.Transitions = append(state.Transitions, *p.currentTransition)
		p.currentTransition = nil
	case scopeState:
		p.currentState = ""
	case scopeFlow:
		// nothing
	}
	return nil
}

func (p *dslParser) currentScope() dslScope {
	if len(p.stack) == 0 {
		return ""
	}
	return p.stack[len(p.stack)-1]
}

func (p *dslParser) parseTransitionTarget(tokens []string) (string, error) {
	if len(tokens) == 2 {
		return unquoteScriptToken(tokens[1]), nil
	}
	if len(tokens) == 3 && strings.EqualFold(tokens[1], "to") {
		return unquoteScriptToken(tokens[2]), nil
	}
	return "", fmt.Errorf("line %d: transition requires target state: transition <state> or transition to <state>", p.lineNo)
}

func parseAction(actionType string, tokens []string, lineNo int) (Action, error) {
	var action Action
	actionType = strings.ToLower(actionType)
	action.Type = ActionType(actionType)
	selector := Selector{}
	actionArgSeen := false
	wantOnSeen := false

	for _, token := range tokens {
		key, value, ok := parseKeyValue(token)
		if !ok {
			return Action{}, fmt.Errorf("line %d: action argument must be key=value", lineNo)
		}
		actionArgSeen = true
		switch strings.ToLower(key) {
		case "name":
			selector.Name = value
		case "ref":
			selector.Ref = value
		case "automationid", "automation_id", "aid", "automationId":
			selector.AutomationID = value
		case "class", "classname", "className":
			selector.ClassName = value
		case "control", "controltype", "controlType":
			selector.ControlType = value
		case "value":
			action.Value = value
		case "wanton", "want_on", "want":
			v, err := strconv.ParseBool(value)
			if err != nil {
				return Action{}, fmt.Errorf("line %d: wantOn expects boolean", lineNo)
			}
			action.WantOn = v
			wantOnSeen = true
		default:
			return Action{}, fmt.Errorf("line %d: unsupported action field %q", lineNo, key)
		}
	}
	if !isActionTypeSupported(actionType) {
		return Action{}, fmt.Errorf("line %d: unsupported action type %q", lineNo, actionType)
	}
	if action.Type == ActionNone {
		if actionArgSeen {
			return Action{}, fmt.Errorf("line %d: none action does not accept arguments", lineNo)
		}
		return action, nil
	}
	if action.Type != ActionNone {
		action.Selector = selector
	}
	if action.Type == ActionFill && !actionFieldPresent(action) {
		return Action{}, fmt.Errorf("line %d: fill action requires value", lineNo)
	}

	if action.Type != ActionNone && selector == (Selector{}) && action.Type != ActionFill {
		return Action{}, fmt.Errorf("line %d: action %q missing selector fields", lineNo, actionType)
	}
	if action.Type == ActionToggleTo && !wantOnSeen {
		return Action{}, fmt.Errorf("line %d: toggle_to action requires wantOn", lineNo)
	}
	return action, nil
}

func isActionTypeSupported(actionType string) bool {
	switch ActionType(actionType) {
	case ActionNone, ActionClick, ActionDoubleClick, ActionInvoke, ActionFill, ActionFocus, ActionToggle, ActionToggleTo, ActionSelect:
		return true
	default:
		return false
	}
}

func actionFieldPresent(action Action) bool {
	return action.Value != ""
}

func parseSelectorAssignments(tokens []string, selector *Selector, lineNo int) error {
	for _, token := range tokens {
		key, value, ok := parseKeyValue(token)
		if !ok {
			return fmt.Errorf("line %d: selector field must be key=value", lineNo)
		}
		switch strings.ToLower(key) {
		case "name":
			selector.Name = value
		case "ref":
			selector.Ref = value
		case "automationid", "automation_id", "automationId":
			selector.AutomationID = value
		case "classname", "className", "class_name":
			selector.ClassName = value
		case "controltype", "controlType", "control_type":
			selector.ControlType = value
		default:
			return fmt.Errorf("line %d: unsupported selector field %q", lineNo, key)
		}
	}
	return nil
}

func parseOptionalSelectorAssignments(tokens []string, selector *Selector, weight *float64, lineNo int) error {
	for _, token := range tokens {
		key, value, ok := parseKeyValue(token)
		if !ok {
			return fmt.Errorf("line %d: optional selector field must be key=value", lineNo)
		}
		switch strings.ToLower(key) {
		case "weight":
			num, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("line %d: invalid weight %q", lineNo, value)
			}
			*weight = num
		default:
			switch strings.ToLower(key) {
			case "name":
				selector.Name = value
			case "ref":
				selector.Ref = value
			case "automationid", "automation_id", "automationId":
				selector.AutomationID = value
			case "classname", "className", "class_name":
				selector.ClassName = value
			case "controltype", "controlType", "control_type":
				selector.ControlType = value
			default:
				return fmt.Errorf("line %d: unsupported optional field %q", lineNo, key)
			}
		}
	}
	return nil
}

func parseKeyValue(raw string) (string, string, bool) {
	pair := strings.SplitN(raw, "=", 2)
	if len(pair) != 2 {
		return "", "", false
	}
	return strings.TrimSpace(pair[0]), unquoteScriptToken(strings.TrimSpace(pair[1])), true
}

func parseDurationMS(raw string, lineNo int) (int, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.TrimSuffix(value, "ms")
	num, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("line %d: invalid duration %q", lineNo, raw)
	}
	return num, nil
}

func stripScriptComment(line string) string {
	var builder strings.Builder
	var quote rune
	for i := 0; i < len(line); i++ {
		char := rune(line[i])
		if quote == 0 {
			if char == '"' || char == '\'' {
				quote = char
			} else if char == '#' {
				break
			} else if char == '/' && i+1 < len(line) && line[i+1] == '/' {
				break
			}
		} else if char == quote {
			quote = 0
		}
		builder.WriteByte(line[i])
	}
	return builder.String()
}

func splitScriptTokens(line string) []string {
	var tokens []string
	var token strings.Builder
	inQuote := rune(0)
	escape := false

	push := func() {
		if token.Len() == 0 {
			return
		}
		tokens = append(tokens, token.String())
		token.Reset()
	}

	for _, char := range line {
		switch {
		case escape:
			token.WriteRune(char)
			escape = false
		case inQuote != 0:
			if char == '\\' {
				escape = true
			} else if char == inQuote {
				inQuote = 0
			} else {
				token.WriteRune(char)
			}
		case char == '\'' || char == '"':
			inQuote = char
		case char == ' ' || char == '\t':
			push()
		default:
			token.WriteRune(char)
		}
	}
	push()
	return tokens
}

func unquoteScriptToken(token string) string {
	return strings.Trim(token, `"'`)
}
