package ui

import "os/exec"

type Model struct {
	State                 string // "namespace_select", "panel_view"
	Namespaces            []string
	Cursor                int
	SelectedNS            string
	PodCursor             int
	PodsPanel             *Panel
	LogsPanels            []*Panel
	ActivePanel           int
	LogPageIndex          int // Current page index for log panels (0-based, 4 panels per page)
	Width                 int
	Height                int
	Err                   error
	Quit                  bool
	SearchTerm            string // Search term for namespace filtering
	NamespaceWatch        bool   // Auto-refresh namespace list
	DeleteConfirmation    string // Namespace to delete (empty if no confirmation pending)
	DeletingNamespace     string // Namespace currently being deleted
	PodDeleteConfirmation string // Pod to delete (empty if no confirmation pending)
	DeletingPod           string // Pod currently being deleted
	DescribePanel         *Panel // Panel to show describe output
	DescribeTarget        string // Pod currently described
	ServiceIPQuery        string
	ServiceIPResult       []string
	ServiceIPSearching    bool
	ServiceIPInputActive  bool
	ServiceIPErr          error
	NSTotalPages          int
	NSCurrentPage         int
	AvailablePods         []string // List of all pods (for lazy log loading)
}

type Panel struct {
	Title     string
	Content   []string
	MaxLines  int
	ScrollPos int
	UpdateCmd *exec.Cmd
	Watch     bool
}

func InitialModel(searchTerm string) *Model {
	return &Model{
		State:                 "namespace_select",
		Namespaces:            []string{},
		Cursor:                0,
		PodCursor:             0,
		PodsPanel:             nil,
		LogsPanels:            []*Panel{},
		ActivePanel:           0,
		SearchTerm:            searchTerm,
		DeletingNamespace:     "",
		PodDeleteConfirmation: "",
		DeletingPod:           "",
		DescribePanel:         nil,
		DescribeTarget:        "",
		ServiceIPQuery:        "",
		ServiceIPResult:       []string{},
		ServiceIPSearching:    false,
		ServiceIPInputActive:  false,
		ServiceIPErr:          nil,
		NSTotalPages:          1,
		NSCurrentPage:         0,
		AvailablePods:         []string{},
	}
}
