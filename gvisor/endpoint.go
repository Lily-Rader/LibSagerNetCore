package gvisor

import (
	"io"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/buffer"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

var _ stack.LinkEndpoint = (*rwEndpoint)(nil)

// rwEndpoint implements the interface of stack.LinkEndpoint from io.ReadWriter.
type rwEndpoint struct {
	// rw is the io.ReadWriter for reading and writing packets.
	rw io.ReadWriter

	// mtu (maximum transmission unit) is the maximum size of a packet.
	mtu uint32

	dispatcher stack.NetworkDispatcher
}

// Attach launches the goroutine that reads packets from io.ReadWriter and
// dispatches them via the provided dispatcher.
func (e *rwEndpoint) Attach(dispatcher stack.NetworkDispatcher) {
	go e.dispatchLoop()
	e.dispatcher = dispatcher
}

// IsAttached implements stack.LinkEndpoint.IsAttached.
func (e *rwEndpoint) IsAttached() bool {
	return e.dispatcher != nil
}

// dispatchLoop dispatches packets to upper layer.
func (e *rwEndpoint) dispatchLoop() {
	for {
		packet := make([]byte, e.mtu)

		n, err := e.rw.Read(packet)
		if err != nil {
			break
		}

		if !e.IsAttached() {
			continue
		}

		pkb := stack.NewPacketBuffer(stack.PacketBufferOptions{
			Data: buffer.NewVectorisedView(n, []buffer.View{buffer.NewViewFromBytes(packet)}),
		})

		switch header.IPVersion(packet) {
		case header.IPv4Version:
			e.dispatcher.DeliverNetworkPacket("", "", header.IPv4ProtocolNumber, pkb)
		case header.IPv6Version:
			e.dispatcher.DeliverNetworkPacket("", "", header.IPv6ProtocolNumber, pkb)
		}
	}
}

func (e *rwEndpoint) writePacket(pkt *stack.PacketBuffer) tcpip.Error {
	vView := buffer.NewVectorisedView(pkt.Size(), pkt.Views())

	if _, err := e.rw.Write(vView.ToView()); err != nil {
		return &tcpip.ErrInvalidEndpointState{}
	}
	return nil
}

// WritePacket writes packet back into io.ReadWriter.
func (e *rwEndpoint) WritePacket(_ stack.RouteInfo, _ tcpip.NetworkProtocolNumber, pkt *stack.PacketBuffer) tcpip.Error {
	return e.writePacket(pkt)
}

// WritePackets writes packets back into io.ReadWriter.
func (e *rwEndpoint) WritePackets(_ stack.RouteInfo, pkts stack.PacketBufferList, _ tcpip.NetworkProtocolNumber) (int, tcpip.Error) {
	n := 0
	for pkt := pkts.Front(); pkt != nil; pkt = pkt.Next() {
		if err := e.writePacket(pkt); err != nil {
			break
		}
		n++
	}
	return n, nil
}

func (e *rwEndpoint) WriteRawPacket(packetBuffer *stack.PacketBuffer) tcpip.Error {
	return &tcpip.ErrNotSupported{}
}

// MTU implements stack.LinkEndpoint.MTU.
func (e *rwEndpoint) MTU() uint32 {
	return e.mtu
}

// Capabilities implements stack.LinkEndpoint.Capabilities.
func (e *rwEndpoint) Capabilities() stack.LinkEndpointCapabilities {
	return stack.CapabilityNone
}

// MaxHeaderLength returns the maximum size of the link layer header. Given it
// doesn't have a header, it just returns 0.
func (*rwEndpoint) MaxHeaderLength() uint16 {
	return 0
}

// LinkAddress returns the link address of this endpoint.
func (*rwEndpoint) LinkAddress() tcpip.LinkAddress {
	return ""
}

// ARPHardwareType implements stack.LinkEndpoint.ARPHardwareType.
func (*rwEndpoint) ARPHardwareType() header.ARPHardwareType {
	return header.ARPHardwareNone
}

// AddHeader implements stack.LinkEndpoint.AddHeader.
func (e *rwEndpoint) AddHeader(tcpip.LinkAddress, tcpip.LinkAddress, tcpip.NetworkProtocolNumber, *stack.PacketBuffer) {
}

// Wait implements stack.LinkEndpoint.Wait.
func (e *rwEndpoint) Wait() {}
