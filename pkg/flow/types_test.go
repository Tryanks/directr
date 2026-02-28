package flow

import "testing"

func TestDefinitionNormalizeAndValidate(t *testing.T) {
	definition := Definition{
		Target: "Dashboard",
		States: map[string]State{
			"Dashboard": {
				Scene: Scene{Required: []Selector{{Name: "工作台"}}},
			},
			"Login": {
				Scene: Scene{Required: []Selector{{AutomationID: "username"}}},
				Transitions: []Transition{{
					To: "Dashboard",
					Action: Action{
						Type: ActionClick,
						Selector: Selector{
							Name: "登录",
						},
					},
				}},
			},
		},
	}

	definition.Normalize()
	if definition.Name == "" {
		t.Fatalf("expected normalized name")
	}
	if definition.Initial != "Dashboard" {
		t.Fatalf("expected normalized initial state to be lexicographically first, got %q", definition.Initial)
	}
	if err := definition.Validate(); err != nil {
		t.Fatalf("expected valid definition, got %v", err)
	}
}

func TestDefinitionValidateFailure(t *testing.T) {
	definition := Definition{
		Initial: "Login",
		Target:  "Dashboard",
		States: map[string]State{
			"Login": {
				Scene: Scene{Required: []Selector{{Name: "登录"}}},
				Transitions: []Transition{{
					To: "Dashboard",
					Action: Action{
						Type:     ActionFill,
						Selector: Selector{Name: "用户名"},
					},
				}},
			},
		},
	}

	err := definition.Validate()
	if err == nil {
		t.Fatalf("expected validation error")
	}
}
