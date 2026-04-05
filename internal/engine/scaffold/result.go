package scaffold

type Result struct {
	Created []string
	Skipped []string
	Errors  []error
}

func (r *Result) AddCreated(path string) {
	r.Created = append(r.Created, path)
}

func (r *Result) AddSkipped(path string) {
	r.Skipped = append(r.Skipped, path)
}

func (r *Result) AddError(err error) {
	r.Errors = append(r.Errors, err)
}
