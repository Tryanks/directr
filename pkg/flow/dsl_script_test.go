package flow

import (
	"testing"
)

func TestParseDefinitionDSLScript(t *testing.T) {
	definition := `
flow demo-login-flow {
  initial: Login
  target: Dashboard
  maxSteps: 20

  state Login {
    scene {
      required automationId=username
      required automationId=password
      required name="登录" controlType=button
      forbidden name="退出登录"
      optional name=menu-home weight=2
    }

    transition Dashboard {
      click name="登录" controlType=button
      wait 500
    }
  }

  state Dashboard {
    scene {
      required name=工作台
    }
  }
}
`

	flowDef, err := ParseDefinitionDSL([]byte(definition))
	if err != nil {
		t.Fatalf("parse dsl: %v", err)
	}
	if flowDef.Initial != "Login" {
		t.Fatalf("initial expected Login, got %q", flowDef.Initial)
	}
	if flowDef.Target != "Dashboard" {
		t.Fatalf("target expected Dashboard, got %q", flowDef.Target)
	}
	login, ok := flowDef.States["Login"]
	if !ok {
		t.Fatal("missing Login state")
	}
	if len(login.Scene.Required) != 3 {
		t.Fatalf("expected 3 required selectors, got %d", len(login.Scene.Required))
	}
	if len(login.Transitions) != 1 {
		t.Fatalf("expected one transition, got %d", len(login.Transitions))
	}
	if login.Transitions[0].Action.Type != ActionClick {
		t.Fatalf("expected click action, got %q", login.Transitions[0].Action.Type)
	}
}

func TestParseDefinitionDSLRejectsInvalidToken(t *testing.T) {
	_, err := ParseDefinitionDSL([]byte(`flow test {
  initial: Start
  target: End
  state Start {
    scene {
      required badtoken
    }
    transition End {
      click name=ok
    }
  }
  state End {
    scene {
      required name=done
    }
  }
}`))
	if err == nil {
		t.Fatalf("expected parse error")
	}
}
