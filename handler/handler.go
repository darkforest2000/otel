package handler

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
	"wapp/constant"
	"wapp/tractx"
	"wapp/usecase"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
)

type Handler struct {
	usecase *usecase.Usecase
}

func New(usecase *usecase.Usecase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) NewServer(ctx context.Context) *http.Server {

	mux := http.NewServeMux()

	// Полностью переопределим структуру именования трассировки
	handleFunc := func(pattern string, handlerFunc func(http.ResponseWriter, *http.Request)) {
		// НЕ используем otelhttp.NewHandler, который переопределяет наши имена спанов
		// Вместо этого настраиваем только трассировку маршрутов
		handler := otelhttp.WithRouteTag(pattern, h.wrapHandler(handlerFunc, pattern))
		mux.Handle(pattern, handler)
	}

	// Register handlers
	handleFunc("/", h.Root)
	handleFunc("/hello/", h.Hello)
	handleFunc("/query", handleQuery)

	// Создаем собственную обертку для основного обработчика
	instrumentedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r)
	})

	// Настраиваем HTTP сервер с нашим обработчиком
	srv := &http.Server{
		Addr:         ":8080",
		BaseContext:  func(_ net.Listener) context.Context { return ctx },
		ReadTimeout:  time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      instrumentedHandler, // Использование нашей обертки вместо otelhttp.NewHandler
	}

	return srv
}

// handleRoot handles requests to the root path "/"
func (h *Handler) Root(w http.ResponseWriter, r *http.Request) {
	// Corner case: Ensure only "/" is handled, not other paths by default
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	fmt.Fprintln(w, "Hello, World from the Go server!")
}

// handleHello handles requests like "/hello/{name}"
func (h *Handler) Hello(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Corner case: Path parameters - extracting dynamic parts of the URL
	parts := strings.Split(r.URL.Path, "/")
	// Expected path: /hello/{name} -> parts = ["", "hello", "{name}"]
	if len(parts) < 3 || parts[2] == "" {
		http.Error(w, "Bad Request: Missing name in path (e.g., /hello/yourname)", http.StatusBadRequest)
		return
	}
	name := parts[2]

	ctx := tractx.New(r.Context())

	ctx, span, stop := ctx.TracerStart("hello")
	defer stop()

	span.SetAttributes(attribute.String("nameInHandler", name))

	name, err := h.usecase.Hello(ctx, name)
	if err != nil {
		if errors.Is(err, constant.NotFound) {
			span.RecordError(err)
			http.Error(w, "Sorry, user not found", http.StatusBadRequest)
			return
		}
		span.RecordError(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Hello, %s!\n", name)
}

// handleQuery handles requests with query parameters like "/query?key=value"
func handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	// Corner case: Query parameters - accessing key-value pairs
	queryValues := r.URL.Query()
	key := queryValues.Get("key")      // Gets the first value associated with "key"
	multiValue := queryValues["multi"] // Gets all values associated with "multi"

	if key == "" {
		fmt.Fprintln(w, "Query parameter 'key' is missing or empty.")
	} else {
		fmt.Fprintf(w, "Value for 'key': %s\n", key)
	}

	if len(multiValue) > 0 {
		fmt.Fprintf(w, "Values for 'multi': %s\n", strings.Join(multiValue, ", "))
	} else {
		fmt.Fprintln(w, "Query parameter 'multi' is missing.")
	}
}
