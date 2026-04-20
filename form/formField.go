package inputs

type FileConfig struct {
	Accept          string   `json:"accept,omitempty"`          // e.g. ".jpg,.png,.pdf" or "image/*"
	MaxSizeBytes    int64    `json:"maxSizeBytes,omitempty"`    // e.g. 10485760 for 10MB
	MaxFiles        int      `json:"maxFiles,omitempty"`        // e.g. 5 for allowing up to 5 files
	UploadURL       string   `json:"uploadUrl,omitempty"`       // e.g. "/api/upload"
	AcceptMimeTypes []string `json:"acceptMimeTypes,omitempty"` // e.g. ["image/jpeg", "application/pdf"]
}

type FieldAction struct {
	When         Rule        `json:"when"`
	Action       string      `json:"action"` // "clear" | "setRequired" | "setOptional" | "show" | "hide" | "setValue"
	TargetFields []string    `json:"targetFields"`
	Value        interface{} `json:"value,omitempty"`
	ValueRef     string      `json:"valueRef,omitempty"`
}

type FieldValidation struct {
	Min            *int   `json:"min,omitempty"`
	Max            *int   `json:"max,omitempty"`
	MinLength      *int   `json:"minLength,omitempty"`
	MaxLength      *int   `json:"maxLength,omitempty"`
	Pattern        string `json:"pattern,omitempty"`
	PatternMessage string `json:"patternMessage,omitempty"`
	Message        string `json:"message,omitempty"`
}
