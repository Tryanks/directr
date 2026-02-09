//go:build windows

package directr

import "github.com/Tryanks/directr/internal/uiautomation"

func traverseUIElementTreeNoCache(ppv *uiautomation.IUIAutomation, root *uiautomation.IUIAutomationElement) *uiautomation.Element {
	elementArr, err := uiautomation.FindAll(root, uiautomation.CreateTrueCondition(ppv))

	newElement := uiautomation.NewElement(root)
	newElement.Populate(false)

	if err == nil && elementArr != nil {
		defer elementArr.Release()
		arrLen := elementArr.Get_Length()
		for i := 0; i < int(arrLen); i++ {
			elem, _ := elementArr.GetElement(int32(i))
			if elem != nil {
				childElem := traverseUIElementTreeNoCache(ppv, elem)
				newElement.Child = append(newElement.Child, childElem)
			}
		}
	}

	return newElement
}
