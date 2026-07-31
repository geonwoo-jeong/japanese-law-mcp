package legalqueryadoption

import "fmt"

func validateAdoptionGraph(history map[string]manifestDocument) error {
	initialCount := 0
	for adoptionID, document := range history {
		if document.PreviousAdoptionID == nil {
			initialCount++
			continue
		}
		previous := *document.PreviousAdoptionID
		if previous == adoptionID {
			return fmt.Errorf("previousAdoptionId は自己参照できません")
		}
		if _, exists := history[previous]; !exists {
			return fmt.Errorf("previousAdoptionId が history に存在しません")
		}
	}
	if initialCount != 1 {
		return fmt.Errorf("初回 adoption manifest は一件でなければなりません")
	}
	states := make(map[string]uint8, len(history))
	for adoptionID := range history {
		if err := visitAdoptionHistory(adoptionID, history, states); err != nil {
			return err
		}
	}
	return nil
}

func visitAdoptionHistory(
	adoptionID string,
	history map[string]manifestDocument,
	states map[string]uint8,
) error {
	switch states[adoptionID] {
	case 1:
		return fmt.Errorf("adoption history に cycle があります")
	case 2:
		return nil
	}
	states[adoptionID] = 1
	document := history[adoptionID]
	if document.PreviousAdoptionID != nil {
		if err := visitAdoptionHistory(*document.PreviousAdoptionID, history, states); err != nil {
			return err
		}
	}
	states[adoptionID] = 2
	return nil
}
