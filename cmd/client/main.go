package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/netip"
	"os"

	"github.com/alecthomas/kong"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/CobaltCause/go-remote/v2/gen/proto"
)

var CLI struct {
	Address netip.Addr `name:"address" short:"a" required:"" help:"IP address to connect to."`
	Port    uint16     `name:"port" short:"p" required:"" help:"Port to connect to."`

	Start struct {
		Args []string `arg:"" name:"args" help:"Command to start and its arguments." passthrough:"partial"`
	} `cmd:"" help:"Start a remote process."`
	Stop struct {
		Id int `arg:"" name:"id" help:"ID of the remote process to stop."`
	} `cmd:"" help:"Stop a remote process."`
	Status struct {
		Id int `arg:"" name:"id" help:"ID of the remote process to get the status of."`
	} `cmd:"" help:"Get the status of a remote process."`
	Stream struct {
		Id int `arg:"" name:"id" help:"ID of the remote process to stream the output of."`
	} `cmd:"" help:"Stream the output of a remote process."`
}

func main() {
	if err := tryMain(); err != nil {
		log.Print("error: ", err)
		os.Exit(1)
	}
}

func tryMain() error {
	ctx := kong.Parse(&CLI)

	addrPort := netip.AddrPortFrom(CLI.Address, CLI.Port)

	client, err := grpc.NewClient(
		addrPort.String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	grpcClient := pb.NewGoRemoteClient(client)

	switch ctx.Command() {
	case "start <args>":
		// Convert []string to [][]byte.
		args := make([][]byte, len(CLI.Start.Args))
		for i, arg := range CLI.Start.Args {
			args[i] = []byte(arg)
		}

		resp, err := grpcClient.Start(context.Background(), &pb.StartRequest{
			Path: []byte(CLI.Start.Args[0]),
			Args: args,
		})

		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		log.Print("started process with ID: ", resp.GetId())
	case "stop <id>":
		id := uint32(CLI.Stop.Id)
		_, err := grpcClient.Stop(context.Background(), &pb.StopRequest{
			Id: &id,
		})

		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		log.Print("stopped process")
	case "status <id>":
		id := uint32(CLI.Status.Id)
		resp, err := grpcClient.Status(context.Background(), &pb.StatusRequest{
			Id: &id,
		})

		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		log.Print("state: ", resp.GetState())
		if exitCode := resp.ExitCode; exitCode != nil {
			log.Print("exit code: ", *exitCode)
		}
	case "stream <id>":
		id := uint32(CLI.Stream.Id)
		stream, err := grpcClient.Stream(context.Background(), &pb.StatusRequest{
			Id: &id,
		})

		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}

		for {
			resp, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				return fmt.Errorf("failed to receive data: %w", err)
			}

			switch resp.GetKind() {
			case *pb.OutputKind_STDOUT.Enum():
				_, err = os.Stdout.Write(resp.GetData())
				if err != nil {
					return fmt.Errorf("failed to write to stdout: %w", err)
				}
			case *pb.OutputKind_STDERR.Enum():
				_, err = os.Stderr.Write(resp.GetData())
				if err != nil {
					return fmt.Errorf("failed to write to stderr: %w", err)
				}
			default:
				panic("unreachable")
			}
		}
	default:
		panic(ctx.Command())
	}

	return nil
}
