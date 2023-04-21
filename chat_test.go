package main

import (
	"errors"
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func Test_setStatusMsg(t *testing.T) {
	m := model{}

	if m.statusLine != "" {
		t.Error("expecting initial status line to be empty")
	}

	m.setStatusMsg("Oops")
	if m.statusLine != "Oops" {
		t.Errorf("set status line: %s", m.statusLine)
	}
}

func Test_setStatus(t *testing.T) {
	m := model{}

	if m.status != statusAwaitingInput {
		t.Errorf("invalid initial status: %v", m.status)
	}

	m.setStatus(statusAwaitingResponse)
	if m.status != statusAwaitingResponse {
		t.Errorf("set status: %v", m.status)
	}
	if m.statusLine != "... Awaiting response ..." {
		t.Errorf("set status, status line: %s", m.statusLine)
	}

	m.setStatus(statusAwaitingAction)
	if m.status != statusAwaitingAction {
		t.Errorf("set status: %v", m.status)
	}
	if m.statusLine != "Enter command" {
		t.Errorf("set status, status line: %s", m.statusLine)
	}

	m.setStatus(statusAwaitingInput)
	if m.status != statusAwaitingInput {
		t.Errorf("set status: %v", m.status)
	}
	if m.statusLine != "Enter to send, Ctrl+D to quit" {
		t.Errorf("set status, status line: %s", m.statusLine)
	}

	m.setStatus(systemStatus(13))
	if m.status != systemStatus(13) {
		t.Errorf("set status: %v", m.status)
	}
	if m.statusLine != "" {
		t.Errorf("expected message to not reset to empty: %s", m.statusLine)
	}
}

func Test_modelUpdate_WindowSizeMsg(t *testing.T) {
	m := bootChat(options{}, conversation{})

	if m.width != 30 {
		t.Errorf("initial width: %v", m.width)
	}

	x, _ := m.Update(tea.WindowSizeMsg{Height: 13, Width: 12})
	m, _ = x.(model)
	if m.width != 12 {
		t.Errorf("unexpected width: %v", m.width)
	}
	if m.height != 13 {
		t.Errorf("unexpected height: %v", m.height)
	}

	x, _ = m.Update(tea.WindowSizeMsg{Height: 1, Width: 1312})
	m, _ = x.(model)
	if m.width != 120 {
		t.Errorf("unexpected max width: %v", m.width)
	}
	if m.height != 6 {
		t.Errorf("unexpected min height: %v", m.height)
	}
}

func Test_modelUpdate_KeyMsg_CtrlD(t *testing.T) {
	m := bootChat(options{}, conversation{})
	key := tea.Key{Type: tea.KeyCtrlD}

	_, x := m.Update(tea.KeyMsg(key))
	y := tea.Cmd(tea.Quit)
	if reflect.ValueOf(x) != reflect.ValueOf(y) {
		t.Errorf("want %#v, got %#v (%#v)", reflect.ValueOf(y), reflect.ValueOf(x), x)
	}
}

func Test_modelUpdate_KeyMsg_KeyEsc_InChat(t *testing.T) {
	m := bootChat(options{}, conversation{})
	m.setMode(modeChat)
	if m.status != statusAwaitingInput {
		t.Errorf("invalid chat status: %#v", m.status)
	}

	key := tea.Key{Type: tea.KeyEsc}

	x, cmd := m.Update(tea.KeyMsg(key))
	m, _ = x.(model)
	if m.status != statusAwaitingAction {
		t.Error("esc in normal chat should put us in command")
	}
	if cmd != nil {
		t.Errorf("esc should have no cmd: %#v", cmd)
	}

	x, cmd = m.Update(tea.KeyMsg(key))
	m, _ = x.(model)
	if m.status != statusAwaitingInput {
		t.Error("esc in command should put us in normal chat")
	}
	if cmd != nil {
		t.Errorf("esc should have no cmd: %#v", cmd)
	}

	m.setStatus(statusAwaitingResponse)
	x, cmd = m.Update(tea.KeyMsg(key))
	m, _ = x.(model)
	if m.status != statusAwaitingResponse {
		t.Error("esc when awaiting response should do nothing")
	}
	if cmd != nil {
		t.Errorf("esc should have no cmd: %#v", cmd)
	}
}

func Test_modelUpdate_KeyMsg_KeyEsc_OutsideChatRevertsToChat(t *testing.T) {
	m := bootChat(options{}, conversation{})
	m.setMode(modeSelectCode)
	if m.status != statusAwaitingInput {
		t.Errorf("invalid chat status: %#v", m.status)
	}

	key := tea.Key{Type: tea.KeyEsc}

	x, cmd := m.Update(tea.KeyMsg(key))
	m, _ = x.(model)
	if m.status != statusAwaitingInput {
		t.Errorf("invalid post-esc status: %#v", m.status)
	}
	if m.mode != modeChat {
		t.Errorf("esc in selection mode should revert to chat")
	}
	if cmd != nil {
		t.Errorf("esc should have no cmd: %#v", cmd)
	}
}

func Test_modelUpdate_KeyMsg_CtrlQ_NerfsListAction(t *testing.T) {
	m := bootChat(options{}, conversation{})
	key := tea.Key{Type: tea.KeyCtrlQ}

	_, cmd := m.Update(tea.KeyMsg(key))
	if cmd != nil {
		t.Errorf("ctrl+q should do nothing: %#v", cmd)
	}
}

func Test_modelUpdate_KeyMsg_CtrlS_SwitchesToSelection(t *testing.T) {
	m := bootChat(options{}, conversation{})
	key := tea.Key{Type: tea.KeyCtrlS}

	_, cmd := m.Update(tea.KeyMsg(key))
	x := cmd().(tea.BatchMsg)
	if len(x) != 1 {
		t.Errorf("want one msg, got %#v", x)
	}

	if y, ok := x[0]().(executionResult); !ok {
		t.Error("expected execution result")
	} else if y.model.mode != modeSelectCode {
		t.Error("ctrl+S should switch to code select")
	}
}

func Test_modelUpdate_KeyMsg_CtrlC_CopiesFromSelectionInSelectMode(t *testing.T) {
	m := bootChat(options{}, conversation{})
	key := tea.Key{Type: tea.KeyCtrlC}

	m.setMode(modeSelectCode)

	_, cmd := m.Update(tea.KeyMsg(key))
	x := cmd().(tea.BatchMsg)
	if len(x) != 1 {
		t.Errorf("want one msg, got %#v", x)
	}

	if y, ok := x[0]().(executionResult); !ok {
		t.Error("expected execution result")
	} else if y.model.mode != modeChat {
		t.Error("ctrl+S should switch to code select")
	}
}

func Test_modelUpdate_KeyMsg_CtrlC_CopiesFromChatInChatMode(t *testing.T) {
	m := bootChat(options{}, conversation{})
	key := tea.Key{Type: tea.KeyCtrlC}

	_, cmd := m.Update(tea.KeyMsg(key))
	x := cmd().(tea.BatchMsg)
	if len(x) != 1 {
		t.Errorf("want one msg, got %#v", x)
	}

	if y, ok := x[0]().(executionResult); !ok {
		t.Error("expected execution result")
	} else if y.model.mode != modeChat {
		t.Error("ctrl+S should switch to code select")
	}
}

func Test_modelUpdate_KeyMsg_Enter_CopiesSelectionInSelectMode(t *testing.T) {
	m := bootChat(options{}, conversation{})
	key := tea.Key{Type: tea.KeyEnter}

	m.setMode(modeSelectCode)

	_, cmd := m.Update(tea.KeyMsg(key))
	x := cmd().(tea.BatchMsg)
	if len(x) != 1 {
		t.Errorf("want one msg, got %#v", x)
	}

	if y, ok := x[0]().(executionResult); !ok {
		t.Error("expected execution result")
	} else if y.model.mode != modeChat {
		t.Error("ctrl+S should switch to code select")
	}
}

func Test_modelUpdate_KeyMsg_Enter_ChatMode_UpdatesViewportOnEmptyPrompt(t *testing.T) {
	m := bootChat(options{}, conversation{})
	key := tea.Key{Type: tea.KeyEnter}

	_, cmd := m.Update(tea.KeyMsg(key))
	x := cmd().(tea.BatchMsg)
	if len(x) != 1 {
		t.Errorf("want one msg, got %#v", x)
	}

	if _, ok := x[0]().(refresh); !ok {
		t.Error("expected just refresh")
	}
}

func Test_modelUpdate_KeyMsg_Enter_ChatMode_ExecutesActionWhenPromptStartsWithColon(t *testing.T) {
	m := bootChat(options{}, conversation{})
	key := tea.Key{Type: tea.KeyEnter}

	m.prompt.SetValue(":wat")

	_, cmd := m.Update(tea.KeyMsg(key))
	x := cmd().(tea.BatchMsg)
	if len(x) != 1 {
		t.Errorf("want one msg, got %#v", x)
	}

	if y, ok := x[0]().(executionResult); !ok {
		t.Error("expected just refresh")
	} else if y.err == nil {
		t.Error("expected error for unknown command")
	}
}

func Test_modelUpdate_KeyMsg_Enter_ChatMode_FetchesResponse(t *testing.T) {
	m := bootChat(options{}, conversation{})
	key := tea.Key{Type: tea.KeyEnter}

	m.prompt.SetValue("passthrough")

	_, cmd := m.Update(tea.KeyMsg(key))
	x := cmd().(tea.BatchMsg)
	if len(x) != 1 {
		t.Errorf("want one msg, got %#v", x)
	}

	if _, ok := x[0]().(response); !ok {
		t.Error("expected response")
	}
}

func Test_modelUpdate_response_SetsConvo(t *testing.T) {
	m := bootChat(options{}, conversation{})
	c1 := conversation{message{Role: roleGpt, Content: "wat"}}

	m.setStatus(statusAwaitingResponse)
	if len(m.convo) != 0 {
		t.Error("expected empty convo")
	}

	x, _ := m.Update(tea.Msg(response{convo: c1}))
	m, _ = x.(model)

	if m.status != statusAwaitingInput {
		t.Error("expected awaiting input status")
	}
	if len(m.convo) != len(c1) {
		t.Error("expected convo update")
	}
}

func Test_modelUpdate_switchToStatus_SetsStatus(t *testing.T) {
	m := bootChat(options{}, conversation{})

	x, _ := m.Update(tea.Msg(switchToStatus{status: statusAwaitingResponse}))
	m, _ = x.(model)

	if m.status != statusAwaitingResponse {
		t.Error("expected awaiting response status")
	}
}

func Test_modelUpdate_executionResult_SetsStatusMsgOnError(t *testing.T) {
	m := bootChat(options{}, conversation{})
	err := errors.New("say what!")

	x, _ := m.Update(tea.Msg(executionResult{err: err}))
	m, _ = x.(model)

	if m.statusLine != err.Error() {
		t.Error("expected status line update")
	}
}

func Test_modelUpdate_executionResult_UpdatesModelOnSuccess(t *testing.T) {
	m := bootChat(options{}, conversation{})

	m1 := bootChat(options{}, conversation{})
	m1.statusLine = "this is updated model"

	x, _ := m.Update(tea.Msg(executionResult{model: m1}))
	m, _ = x.(model)

	if m.statusLine != "this is updated model" {
		t.Error("expected model update")
	}
}
