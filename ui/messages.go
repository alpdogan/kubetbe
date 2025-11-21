package ui

import "kubetbe/msg"

// Re-export message types for convenience
type TickMsg = msg.TickMsg
type PodUpdateMsg = msg.PodUpdateMsg
type LogUpdateMsg = msg.LogUpdateMsg
type NamespaceListMsg = msg.NamespaceListMsg
type NamespaceDeleteMsg = msg.NamespaceDeleteMsg
type PodDeleteMsg = msg.PodDeleteMsg
type PodDescribeMsg = msg.PodDescribeMsg
type ServiceLookupMsg = msg.ServiceLookupMsg
type ErrorMsg = msg.ErrorMsg
type StartLogLoadMsg = msg.StartLogLoadMsg
