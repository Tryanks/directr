//go:build windows
// +build windows

package directr

import (
	"strings"

	"github.com/Tryanks/directr/internal/uiautomation"
)

type ControlTypeId int32

const (
	controlTypeButton       = ControlTypeId(uiautomation.UIA_ButtonControlTypeId)
	controlTypeEdit         = ControlTypeId(uiautomation.UIA_EditControlTypeId)
	controlTypeText         = ControlTypeId(uiautomation.UIA_TextControlTypeId)
	controlTypeImage        = ControlTypeId(uiautomation.UIA_ImageControlTypeId)
	controlTypeWindow       = ControlTypeId(uiautomation.UIA_WindowControlTypeId)
	controlTypePane         = ControlTypeId(uiautomation.UIA_PaneControlTypeId)
	controlTypeGroup        = ControlTypeId(uiautomation.UIA_GroupControlTypeId)
	controlTypeCustom       = ControlTypeId(uiautomation.UIA_CustomControlTypeId)
	controlTypeDocument     = ControlTypeId(uiautomation.UIA_DocumentControlTypeId)
	controlTypeList         = ControlTypeId(uiautomation.UIA_ListControlTypeId)
	controlTypeListItem     = ControlTypeId(uiautomation.UIA_ListItemControlTypeId)
	controlTypeMenuBar      = ControlTypeId(uiautomation.UIA_MenuBarControlTypeId)
	controlTypeMenu         = ControlTypeId(uiautomation.UIA_MenuControlTypeId)
	controlTypeMenuItem     = ControlTypeId(uiautomation.UIA_MenuItemControlTypeId)
	controlTypeTab          = ControlTypeId(uiautomation.UIA_TabControlTypeId)
	controlTypeTabItem      = ControlTypeId(uiautomation.UIA_TabItemControlTypeId)
	controlTypeTree         = ControlTypeId(uiautomation.UIA_TreeControlTypeId)
	controlTypeTreeItem     = ControlTypeId(uiautomation.UIA_TreeItemControlTypeId)
	controlTypeCheckBox     = ControlTypeId(uiautomation.UIA_CheckBoxControlTypeId)
	controlTypeRadioButton  = ControlTypeId(uiautomation.UIA_RadioButtonControlTypeId)
	controlTypeComboBox     = ControlTypeId(uiautomation.UIA_ComboBoxControlTypeId)
	controlTypeSlider       = ControlTypeId(uiautomation.UIA_SliderControlTypeId)
	controlTypeProgressBar  = ControlTypeId(uiautomation.UIA_ProgressBarControlTypeId)
	controlTypeScrollBar    = ControlTypeId(uiautomation.UIA_ScrollBarControlTypeId)
	controlTypeTable        = ControlTypeId(uiautomation.UIA_TableControlTypeId)
	controlTypeDataGrid     = ControlTypeId(uiautomation.UIA_DataGridControlTypeId)
	controlTypeDataItem     = ControlTypeId(uiautomation.UIA_DataItemControlTypeId)
	controlTypeHeader       = ControlTypeId(uiautomation.UIA_HeaderControlTypeId)
	controlTypeHeaderItem   = ControlTypeId(uiautomation.UIA_HeaderItemControlTypeId)
	controlTypeStatusBar    = ControlTypeId(uiautomation.UIA_StatusBarControlTypeId)
	controlTypeToolBar      = ControlTypeId(uiautomation.UIA_ToolBarControlTypeId)
	controlTypeToolTip      = ControlTypeId(uiautomation.UIA_ToolTipControlTypeId)
	controlTypeTitleBar     = ControlTypeId(uiautomation.UIA_TitleBarControlTypeId)
	controlTypeHyperlink    = ControlTypeId(uiautomation.UIA_HyperlinkControlTypeId)
	controlTypeCalendar     = ControlTypeId(uiautomation.UIA_CalendarControlTypeId)
	controlTypeSpinner      = ControlTypeId(uiautomation.UIA_SpinnerControlTypeId)
	controlTypeSplitButton  = ControlTypeId(uiautomation.UIA_SplitButtonControlTypeId)
	controlTypeAppBar       = ControlTypeId(uiautomation.UIA_AppBarControlTypeId)
	controlTypeSemanticZoom = ControlTypeId(uiautomation.UIA_SemanticZoomControlTypeId)
	controlTypeSeparator    = ControlTypeId(uiautomation.UIA_SeparatorControlTypeId)
	controlTypeThumb        = ControlTypeId(uiautomation.UIA_ThumbControlTypeId)
)

func ControlTypeToName(controlType ControlTypeId) string {
	switch controlType {
	case controlTypeButton:
		return "button"
	case controlTypeEdit:
		return "textbox"
	case controlTypeText:
		return "text"
	case controlTypeImage:
		return "img"
	case controlTypeWindow:
		return "window"
	case controlTypePane:
		return "generic"
	case controlTypeGroup:
		return "generic"
	case controlTypeCustom:
		return "generic"
	case controlTypeDocument:
		return "document"
	case controlTypeList:
		return "list"
	case controlTypeListItem:
		return "listitem"
	case controlTypeMenuBar:
		return "menubar"
	case controlTypeMenu:
		return "menu"
	case controlTypeMenuItem:
		return "menuitem"
	case controlTypeTab:
		return "tablist"
	case controlTypeTabItem:
		return "tab"
	case controlTypeTree:
		return "tree"
	case controlTypeTreeItem:
		return "treeitem"
	case controlTypeCheckBox:
		return "checkbox"
	case controlTypeRadioButton:
		return "radio"
	case controlTypeComboBox:
		return "combobox"
	case controlTypeSlider:
		return "slider"
	case controlTypeProgressBar:
		return "progressbar"
	case controlTypeScrollBar:
		return "scrollbar"
	case controlTypeTable:
		return "table"
	case controlTypeDataGrid:
		return "grid"
	case controlTypeDataItem:
		return "row"
	case controlTypeHeader:
		return "header"
	case controlTypeHeaderItem:
		return "columnheader"
	case controlTypeStatusBar:
		return "statusbar"
	case controlTypeToolBar:
		return "toolbar"
	case controlTypeToolTip:
		return "tooltip"
	case controlTypeTitleBar:
		return "titlebar"
	case controlTypeHyperlink:
		return "link"
	case controlTypeCalendar:
		return "calendar"
	case controlTypeSpinner:
		return "spinbutton"
	case controlTypeSplitButton:
		return "splitbutton"
	case controlTypeAppBar:
		return "appbar"
	case controlTypeSemanticZoom:
		return "semanticzoom"
	case controlTypeSeparator:
		return "separator"
	case controlTypeThumb:
		return "thumb"
	default:
		return ""
	}
}

func ControlTypeFromName(name string) (ControlTypeId, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "button":
		return controlTypeButton, true
	case "textbox":
		return controlTypeEdit, true
	case "text":
		return controlTypeText, true
	case "img", "image":
		return controlTypeImage, true
	case "window":
		return controlTypeWindow, true
	case "document":
		return controlTypeDocument, true
	case "list":
		return controlTypeList, true
	case "listitem":
		return controlTypeListItem, true
	case "menubar":
		return controlTypeMenuBar, true
	case "menu":
		return controlTypeMenu, true
	case "menuitem":
		return controlTypeMenuItem, true
	case "tablist":
		return controlTypeTab, true
	case "tab":
		return controlTypeTabItem, true
	case "tree":
		return controlTypeTree, true
	case "treeitem":
		return controlTypeTreeItem, true
	case "checkbox":
		return controlTypeCheckBox, true
	case "radio":
		return controlTypeRadioButton, true
	case "combobox":
		return controlTypeComboBox, true
	case "slider":
		return controlTypeSlider, true
	case "progressbar":
		return controlTypeProgressBar, true
	case "scrollbar":
		return controlTypeScrollBar, true
	case "table":
		return controlTypeTable, true
	case "grid":
		return controlTypeDataGrid, true
	case "row":
		return controlTypeDataItem, true
	case "header":
		return controlTypeHeader, true
	case "columnheader":
		return controlTypeHeaderItem, true
	case "statusbar":
		return controlTypeStatusBar, true
	case "toolbar":
		return controlTypeToolBar, true
	case "tooltip":
		return controlTypeToolTip, true
	case "titlebar":
		return controlTypeTitleBar, true
	case "link":
		return controlTypeHyperlink, true
	case "calendar":
		return controlTypeCalendar, true
	case "spinbutton":
		return controlTypeSpinner, true
	case "splitbutton":
		return controlTypeSplitButton, true
	case "appbar":
		return controlTypeAppBar, true
	case "semanticzoom":
		return controlTypeSemanticZoom, true
	case "separator":
		return controlTypeSeparator, true
	case "thumb":
		return controlTypeThumb, true
	default:
		return 0, false
	}
}
