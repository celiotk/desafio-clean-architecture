package main

import (
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"time"

	graphql_handler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/celiotk/desafio-clean-architecture/configs"
	"github.com/celiotk/desafio-clean-architecture/internal/event/handler"
	"github.com/celiotk/desafio-clean-architecture/internal/infra/graph"
	"github.com/celiotk/desafio-clean-architecture/internal/infra/grpc/pb"
	"github.com/celiotk/desafio-clean-architecture/internal/infra/grpc/service"
	"github.com/celiotk/desafio-clean-architecture/internal/infra/web/webserver"
	"github.com/celiotk/desafio-clean-architecture/pkg/events"
	"github.com/streadway/amqp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	// mysql
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	configs, err := configs.LoadConfig(".")
	if err != nil {
		panic(err)
	}

	db, err := sql.Open(configs.DBDriver, fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", configs.DBUser, configs.DBPassword, configs.DBHost, configs.DBPort, configs.DBName))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rabbitMQChannel := getRabbitMQChannel(configs.RabbitMQHost, configs.RabbitMQPort)

	eventDispatcher := events.NewEventDispatcher()
	eventDispatcher.Register("OrderCreated", &handler.OrderCreatedHandler{
		RabbitMQChannel: rabbitMQChannel,
	})

	createOrderUseCase := NewCreateOrderUseCase(db, eventDispatcher)
	listOrderUseCase := NewListOrderUseCase(db)

	webserver := webserver.NewWebServer(configs.WebServerPort)
	webOrderHandler := NewWebOrderHandler(db, eventDispatcher)
	webserver.AddHandler("/order", webOrderHandler.Create, http.MethodPost)
	webserver.AddHandler("/order", webOrderHandler.List, http.MethodGet)
	fmt.Println("Starting web server on port", configs.WebServerPort)
	go webserver.Start()

	grpcServer := grpc.NewServer()
	createOrderService := service.NewOrderService(*createOrderUseCase, *listOrderUseCase)
	pb.RegisterOrderServiceServer(grpcServer, createOrderService)
	reflection.Register(grpcServer)

	fmt.Println("Starting gRPC server on port", configs.GRPCServerPort)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", configs.GRPCServerPort))
	if err != nil {
		panic(err)
	}
	go grpcServer.Serve(lis)

	srv := graphql_handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{
		CreateOrderUseCase: *createOrderUseCase,
		ListOrderUseCase:   *listOrderUseCase,
	}}))
	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	fmt.Println("Starting GraphQL server on port", configs.GraphQLServerPort)
	http.ListenAndServe(":"+configs.GraphQLServerPort, nil)
}

func getRabbitMQChannel(host string, port string) *amqp.Channel {
	fmt.Println("Connecting to RabbitMQ on port", port)
	var conn *amqp.Connection
	var err error
	for attempt := 0; attempt < 20; attempt++ {
		conn, err = amqp.Dial(fmt.Sprintf("amqp://guest:guest@%s:%s/", host, port))
		if err == nil {
			fmt.Println("Connected to RabbitMQ")
			break
		}
		fmt.Println("Failed to connect to RabbitMQ. Retrying in 5 seconds")
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		panic(err)
	}
	ch, err := conn.Channel()
	if err != nil {
		panic(err)
	}
	return ch
}
