package msg

type NamespaceListMsg struct {
	Namespaces []string
}

type NamespaceDeleteMsg struct {
	Namespace string
	Err       error
}

type PodDeleteMsg struct {
	Namespace string
	Pod       string
	Err       error
}

type PodDescribeMsg struct {
	Namespace string
	Pod       string
	Content   []string
	Err       error
}

type ServiceLookupMsg struct {
	IP     string
	Result []string
	Err    error
}

type ErrorMsg struct {
	Err error
}

type PodUpdateMsg struct {
	Content []string
	Err     error
}

type LogUpdateMsg struct {
	PodName string
	Content []string
	Err     error
}

type TickMsg struct{}

type StartLogLoadMsg struct {
	PodName string
}
