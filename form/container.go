package inputs

type Container struct {
	Key            string      `json:"id"`
	Direction      string      `json:"direction"` // horizontal, vertical
	Gap            int         `json:"gap"`       //
	Card           bool        `json:"card,omitempty"`
	Align          string      `json:"align"` // "start" | "center" | "end" | "between" - оба стороны | "stretch" - растянуть по всей высоте
	GridColumns    int         `json:"gridColumns,omitempty"`
	Title          string      `json:"title"`
	Fields         []Input     `json:"fields,omitempty"`
	Containers     []Container `json:"containers,omitempty"`
	VisibilityRule *Rule       `json:"visibilityRule,omitempty"`
}

// GetContainerByKey ищет контейнер по ключу в текущем контейнере и его потомках (рекурсивно)
func (c *Container) GetContainerByKey(key string) *Container {
	if c.Key == key {
		return c
	}
	for i := range c.Containers {
		if result := c.Containers[i].GetContainerByKey(key); result != nil {
			return result
		}
	}
	return nil
}

// GetContainerByKeyInSlice ищет контейнер по ключу в срезе контейнеров
func GetContainerByKeyInSlice(containers []Container, key string) *Container {
	for i := range containers {
		if result := containers[i].GetContainerByKey(key); result != nil {
			return result
		}
	}
	return nil
}
