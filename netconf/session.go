package netconf

import "encoding/xml"

// Session defines the necessary components for a Netconf session
type Session struct {
	Transport          Transport
	SessionID          int
	ServerCapabilities []string
	ErrOnWarning       bool
	log                Logger
}

// Close is used to close and end a transport session
func (s *Session) Close() error {
	return s.Transport.Close()
}

// Exec is used to execute an RPC method or methods with the supplied message-id.
func (s *Session) Exec(messageID string, methods ...RPCMethod) (*RPCReply, error) {
	rpc := NewRPCMessage(methods)
	rpc.MessageID = messageID

	request, err := xml.Marshal(rpc)
	if err != nil {
		return nil, err
	}

	header := []byte(xml.Header)
	request = append(header, request...)

	s.log.Debugf("Exec: REQUEST: %s\n", request)

	rawXML, err := s.Transport.SendReceive(messageID, request)
	if err != nil {
		return nil, err
	}
	s.log.Debugf("Exec: REPLY: %s\n", rawXML)

	reply := &RPCReply{}
	reply.RawReply = string(rawXML)

	if err := xml.Unmarshal(rawXML, reply); err != nil {
		return nil, err
	}

	if reply.Errors != nil {
		// We have errors, lets see if it's a warning or an error.
		for _, rpcErr := range reply.Errors {
			if rpcErr.Severity == "error" || s.ErrOnWarning {
				return reply, &rpcErr
			}
		}

	}

	return reply, nil
}

// NewSession creates a new NetConf session using the provided transport layer.
func NewSession(t Transport, log Logger) *Session {
	s := new(Session)
	s.Transport = t
	s.log = log

	// Receive Servers Hello message
	serverHello, _ := t.ReceiveHello()
	s.SessionID = serverHello.SessionID
	s.ServerCapabilities = serverHello.Capabilities

	// Send our hello using default capabilities.
	t.SendHello(&HelloMessage{Capabilities: DefaultCapabilities})

	t.StartReader()

	return s
}
