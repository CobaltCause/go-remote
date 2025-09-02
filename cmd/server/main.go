package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/netip"
	"os"
	"sync"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/CobaltCause/go-remote/v2/gen/proto"
	pm "github.com/CobaltCause/go-remote/v2/pkg"
	"github.com/alecthomas/kong"
)

type grpcServer struct {
	pb.UnimplementedGoRemoteServer

	processManager pm.ProcessManager
}

func (s *grpcServer) Start(_ context.Context, req *pb.StartRequest) (
	*pb.StartResponse, error,
) {
	// Convert [][]byte to []string.
	args := make([]string, len(req.GetArgs()))
	for i, arg := range req.GetArgs() {
		args[i] = string(arg)
	}

	id, err := s.processManager.Start(string(req.GetPath()), args...)

	if err != nil {
		log.Print("failed to start process: ", err)

		// Mostly likely the error has to do with the specific value the client
		// supplied, e.g. a program not being found. Ideally we'd also detect
		// other error conditions and use an appopriate status code.
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	log.Print("started process with ID: ", id)

	id_uint32 := uint32(id)

	return &pb.StartResponse{
			Id: &id_uint32,
		},
		nil
}

func (s *grpcServer) Stop(_ context.Context, req *pb.StopRequest) (
	*emptypb.Empty, error,
) {
	err := s.processManager.Stop(int(req.GetId()))

	if err != nil {
		log.Print("failed to stop process: ", err)

		if errors.Is(err, pm.ErrUnknownID) {
			return nil, status.Error(codes.NotFound, err.Error())
		} else {
			return nil, err
		}
	}

	log.Print("stopped process ID: ", req.GetId())

	return &emptypb.Empty{}, nil
}

func (s *grpcServer) Status(_ context.Context, req *pb.StatusRequest) (
	*pb.StatusResponse, error,
) {

	st, err := s.processManager.Status(int(req.GetId()))

	if err != nil {
		log.Print("failed to get process status: ", err)
		return nil, status.Error(codes.NotFound, err.Error())
	}

	log.Print("responding with state of process ID: ", req.GetId())

	switch st.State {
	case pm.EXITED:
		exitCode := int32(st.ExitCode)
		return &pb.StatusResponse{
				ExitCode: &exitCode,
				State:    pb.State_EXITED.Enum(),
			},
			nil
	case pm.RUNNING:
		return &pb.StatusResponse{
				State: pb.State_RUNNING.Enum(),
			},
			nil
	case pm.STOPPED:
		return &pb.StatusResponse{
				State: pb.State_STOPPED.Enum(),
			},
			nil
	default:
		panic("unreachable")
	}
}

func (s *grpcServer) Stream(
	req *pb.StatusRequest,
	stream grpc.ServerStreamingServer[pb.Output],
) error {
	stdout, stderr := s.processManager.Stream(int(req.GetId()))

	if stdout == nil || stderr == nil {
		log.Print("failed to stream process output: process ID not found")
		return status.Error(codes.NotFound, "process ID not found")
	}

	log.Print("streaming output of process ID: ", req.GetId())

	outputCh := make(chan *pb.Output)
	var wg sync.WaitGroup
	var sendBreak atomic.Bool

	handleStream := func(reader io.ReadCloser, kind pb.OutputKind) {
		defer wg.Done()
		defer func() {
			if reader.Close() != nil {
				panic("unreachable")
			}
		}()

		buf := make([]byte, 1024)

		for {
			if sendBreak.Load() {
				break
			}

			n, err := reader.Read(buf)

			if errors.Is(err, io.EOF) {
				log.Print("process output finished streaming: ", kind)
				break
			} else if err != nil {
				panic("unreachable")
			}

			outputCh <- &pb.Output{
				Data: buf[:n],
				Kind: &kind,
			}
		}
	}

	wg.Add(3)
	go handleStream(stdout, *pb.OutputKind_STDOUT.Enum())
	go handleStream(stderr, *pb.OutputKind_STDERR.Enum())
	go func() {
		defer wg.Done()

		for {
			if err := stream.Send(<-outputCh); err != nil {
				log.Print("failed to stream output: ", err)
				sendBreak.Store(true)
				break
			}
		}
	}()
	wg.Wait()

	return nil
}

var CLI struct {
	Address netip.Addr `name:"address" short:"a" required:"" help:"IP address to listen on."`
	Port    uint16     `name:"port" short:"p" required:"" help:"Port to listen on."`
}

func main() {
	if err := tryMain(); err != nil {
		log.Print("error: ", err)
		os.Exit(1)
	}
}

func tryMain() error {
	kong.Parse(&CLI)

	addrPort := netip.AddrPortFrom(CLI.Address, CLI.Port)
	listener, err := net.Listen("tcp", addrPort.String())
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	log.Print("listening")

	server := grpc.NewServer()
	pb.RegisterGoRemoteServer(server, &grpcServer{})

	if err := server.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}
