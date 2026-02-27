package main

import (
    "context"
    "errors"
    "fmt"
    "os"
    "os/signal"
    "slices"
    "strconv"
    "strings"
    "sync"
    "syscall"

    "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
    "github.com/levelshatter/awsum/commands"
    "github.com/levelshatter/awsum/internal/app"
    "github.com/levelshatter/awsum/service"
    "github.com/urfave/cli/v3"
)

func main() {
    resources := app.Setup()

    var once sync.Once

    cleanup := func() {
        once.Do(func() {
            if cleanupErrors := resources.Cleanup(); len(cleanupErrors) > 0 {
                for _, err := range cleanupErrors {
                    fmt.Printf("failed to cleanup local awsum resource: %s\n", err)
                }
            }
        })
    }

    exit := make(chan os.Signal, 1)
    signal.Notify(exit, os.Kill, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

    defer func() {
        signal.Stop(exit)
        cleanup()
    }()

    cmd := &cli.Command{
        Name:        "awsum",
        Usage:       "a fun CLI tool for working with AWS infra",
        Description: "awsum allows you to rapidly develop with your own AWS infra via the command line",
        Commands: []*cli.Command{
            {
                Name: "configure",
                Action: func(ctx context.Context, command *cli.Command) error {
                    return commands.Configure()
                },
            },
            {
                Name: "instance",
                Commands: []*cli.Command{
                    {
                        Name:  "list",
                        Usage: "display a formatted list of Info instances",
                        Flags: []cli.Flag{
                            &cli.StringFlag{
                                Name:     "format",
                                Usage:    "pretty|csv",
                                Value:    "pretty",
                                OnlyOnce: true,
                                Validator: func(s string) error {
                                    if s != "pretty" && s != "csv" {
                                        return errors.New("invalid format, must be pretty or csv")
                                    }

                                    return nil
                                },
                                ValidateDefaults: true,
                            },
                        },
                        Action: func(ctx context.Context, command *cli.Command) error {
                            return commands.InstanceList(ctx, command.String("format"))
                        },
                    },
                    {
                        Name:    "shell",
                        Usage:   "run a command or start a shell (via SSH) on ec2 instance(s) matched by the given filters",
                        Suggest: true,
                        Flags: []cli.Flag{
                            &cli.StringFlag{
                                Name:     "user",
                                Aliases:  []string{"as"},
                                Usage:    "which ssh user to connect as",
                                Value:    "ec2-user",
                                OnlyOnce: true,
                            },
                            &cli.StringFlag{
                                Name:     "name",
                                Aliases:  []string{"n"},
                                Usage:    "a fuzzy filter that matches against ec2 instance names (from tags)",
                                OnlyOnce: true,
                            },
                            &cli.BoolFlag{
                                Name:     "quiet",
                                Aliases:  []string{"q"},
                                Usage:    "whether to disable the additional debug information when a shell starts and ends",
                                Value:    false,
                                OnlyOnce: true,
                            },
                            &cli.BoolFlag{
                                Name:     "parallel",
                                Aliases:  []string{"p"},
                                Usage:    "whether to run the commands in parallel across instances. note: parallel output has no debug information.",
                                Value:    false,
                                OnlyOnce: true,
                            },
                        },
                        Action: func(ctx context.Context, command *cli.Command) error {
                            return commands.InstanceShell(commands.InstanceShellOptions{
                                Ctx: ctx,
                                InstanceFilters: service.InstanceFilters{
                                    Name: command.String("name"),
                                },
                                User:     command.String("user"),
                                Command:  strings.Join(command.Args().Slice(), " "),
                                Quiet:    command.Bool("quiet"),
                                Parallel: command.Bool("parallel"),
                            })
                        },
                    },
                    {
                        Name:    "load-balance",
                        Usage:   "create or update load balancer resources for a service on desired instances",
                        Suggest: true,
                        Flags: []cli.Flag{
                            &cli.StringFlag{
                                Name:     "service",
                                Usage:    "the name of the new or existing service you wish to load-balance",
                                OnlyOnce: true,
                                Required: true,
                            },
                            &cli.StringFlag{
                                Name:     "name",
                                Usage:    "a fuzzy filter that matches against ec2 instance names (from tags) to include in the load-balance resource creation",
                                OnlyOnce: true,
                            },
                            &cli.StringFlag{
                                Name:     "port",
                                Usage:    "the port to create the load-balancer listener on & the target traffic port of your service on each instance",
                                OnlyOnce: true,
                                Required: true,
                                Validator: func(s string) error {
                                    parts := strings.Split(s, ":")

                                    if len(parts) != 2 {
                                        return errors.New("must be in format <load balancer port>:<instance port>")
                                    }

                                    return nil
                                },
                            },
                            &cli.StringFlag{
                                Name:     "protocol",
                                Usage:    "the network protocol of the traffic your service uses",
                                OnlyOnce: true,
                                Required: true,
                                Value:    "http:http",
                                Validator: func(s string) error {
                                    parts := strings.Split(s, ":")

                                    if len(parts) != 2 {
                                        return errors.New("must be in format <load balancer listener protocol>:<service traffic protocol>")
                                    }

                                    listenerProtocol := parts[0]
                                    trafficProtocol := parts[1]

                                    if !slices.Contains(types.ProtocolEnum("").Values(), types.ProtocolEnum(strings.ToUpper(listenerProtocol))) {
                                        return errors.New("invalid listener protocol")
                                    }

                                    if !slices.Contains(types.ProtocolEnum("").Values(), types.ProtocolEnum(strings.ToUpper(trafficProtocol))) {
                                        return errors.New("invalid traffic protocol")
                                    }

                                    return nil
                                },
                                ValidateDefaults: true,
                            },
                            &cli.StringFlag{
                                Name:        "ip-protocol",
                                DefaultText: "tcp",
                                Usage:       "the underlying IP protocol for your service's load-balancer",
                                Value:       "tcp",
                                OnlyOnce:    true,
                            },
                            &cli.StringSliceFlag{
                                Name:     "certificate",
                                Usage:    "the ACM certificate name(s) to attach to the created listener. the first certificate is default.",
                                OnlyOnce: false,
                            },
                            &cli.StringSliceFlag{
                                Name:     "domain",
                                Usage:    "the FQDN(s) to attach to the desired load balancer.",
                                OnlyOnce: false,
                            },
                            &cli.BoolFlag{
                                Name:     "private",
                                Usage:    "if your load balancer and domain records should be private (not visible to the internet)",
                                OnlyOnce: true,
                            },
                        },
                        Action: func(ctx context.Context, command *cli.Command) error {
                            portParts := strings.Split(command.String("port"), ":")

                            lbPort, err := strconv.ParseInt(portParts[0], 10, 32)

                            if err != nil {
                                return err
                            }

                            trafficPort, err := strconv.ParseInt(portParts[1], 10, 32)

                            if err != nil {
                                return err
                            }

                            protocolParts := strings.Split(command.String("protocol"), ":")

                            return commands.InstanceLoadBalance(commands.InstanceLoadBalanceOptions{
                                Ctx:         ctx,
                                ServiceName: command.String("service"),
                                InstanceFilters: service.InstanceFilters{
                                    Name: command.String("name"),
                                },
                                LoadBalancerPort:             int32(lbPort),
                                LoadBalancerIpProtocol:       strings.ToLower(command.String("ip-protocol")),
                                LoadBalancerListenerProtocol: types.ProtocolEnum(strings.ToUpper(protocolParts[0])),
                                TrafficPort:                  int32(trafficPort),
                                TrafficProtocol:              types.ProtocolEnum(strings.ToUpper(protocolParts[1])),
                                CertificateNames:             command.StringSlice("certificate"),
                                DomainNames:                  command.StringSlice("domain"),
                                Private:                      command.Bool("private"),
                            })
                        },
                    },
                },
            },
        },
    }

    if err := cmd.Run(app.Ctx, os.Args); err != nil {
        fmt.Printf("failed to run command: %s\n", err)
        os.Exit(1)
    }

    close(exit)
    cleanup()
}
