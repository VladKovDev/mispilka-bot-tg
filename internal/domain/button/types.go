package button

// ButtonRegistry manages shared button sets
type ButtonRegistry struct {
	ButtonSets map[string]*ButtonSet `json:"button_sets"`
}

// ButtonSet represents a set of buttons
type ButtonSet struct {
	Rows []ButtonRow `json:"rows"`
}

// ButtonRow represents a row of buttons
type ButtonRow struct {
	Buttons []Button `json:"buttons"`
}

// Button represents a single button
type Button struct {
	Type     string `json:"type"`     // url, callback
	Text     string `json:"text"`
	URL      string `json:"url,omitempty"`
	Callback string `json:"callback,omitempty"`
}

// NewButtonRegistry creates a new button registry
func NewButtonRegistry() *ButtonRegistry {
	return &ButtonRegistry{
		ButtonSets: make(map[string]*ButtonSet),
	}
}

// Get retrieves a button set by reference
func (r *ButtonRegistry) Get(ref string) (*ButtonSet, bool) {
	bs, ok := r.ButtonSets[ref]
	return bs, ok
}

// Set stores a button set
func (r *ButtonRegistry) Set(ref string, bs *ButtonSet) {
	r.ButtonSets[ref] = bs
}

// Delete removes a button set
func (r *ButtonRegistry) Delete(ref string) {
	delete(r.ButtonSets, ref)
}
