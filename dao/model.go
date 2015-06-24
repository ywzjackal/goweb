package dao

type Model struct {
	primarysOld []*modelField // Primary key Fields of OldValue
}

func (m *Model) IsNew() bool {
	return m.primarysOld == nil
}
