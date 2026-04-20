package inputs

type Form struct {
	Containers  *[]Container  `json:"containers"`
	FormActions *[]FormAction `json:"actions,omitempty"`
}
