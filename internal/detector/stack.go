package detector

type Stack struct {
	ID          string
	Name        string
	Backend     string
	Frontend    string
	Framework   string
	HasDB       bool
	HasPubSub   bool
	HasAuth     bool
	HasDocker   bool
	HasCI       bool
	GoModule    string
	NodePkgName string
	Confidence  float64
}
