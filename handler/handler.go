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
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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

// httpErrorHelper is a refactored helper to set span status and record errors correctly.
func httpErrorHelper(span trace.Span, w http.ResponseWriter, r *http.Request, httpStatus int, publicMessage string, internalError error) {
	// Always set standard HTTP attributes. These are captured by h.wrapHandler at the end,
	// but setting them here can be useful if the handler exits early.
	// The wrapHandler will set the final authoritative one.
	span.SetAttributes(attribute.Int("http.status_code", httpStatus))

	// For SpanKind.SERVER:
	// - 5xx status codes indicate a server error, so set SpanStatus to Error.
	// - 4xx status codes indicate a client error. The server handled it correctly,
	//   so SpanStatus should NOT be Error. We can still record the error for diagnostics.
	if httpStatus >= 500 {
		span.SetStatus(codes.Error, publicMessage)
		span.SetAttributes(attribute.String("http.res", "error"))
	} else if httpStatus >= 400 && httpStatus < 500 {
		span.SetAttributes(attribute.String("http.res", "client_error"))
	}

	if internalError != nil {
		span.RecordError(internalError)
	} else {
		// Create a generic error if no specific one is provided
		span.RecordError(errors.New(publicMessage))
	}

	http.Error(w, publicMessage, httpStatus)
}

// handleHello handles requests like "/hello/{name}"
func (h *Handler) Hello(w http.ResponseWriter, r *http.Request) {
	ctx := tractx.New(r.Context())
	span := trace.SpanFromContext(ctx) // Get the span from the context (created by wrapHandler)

	if r.Method != http.MethodGet {
		httpErrorHelper(span, w, r, http.StatusMethodNotAllowed, "Method Not Allowed", errors.New("method not allowed"))
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 || parts[2] == "" {
		errDetail := fmt.Errorf("bad request: missing name in path (e.g., /hello/yourname)")
		httpErrorHelper(span, w, r, http.StatusBadRequest, "Bad Request: Missing name in path (e.g., /hello/yourname)", errDetail)
		return
	}
	name := parts[2]

	span.SetAttributes(attribute.String("handler.input.name", name))

	// Call usecase directly. The usecase method itself will create its own span as a child of the current one.
	usecaseResultName, err := h.usecase.Hello(ctx, name)
	if err != nil {
		if errors.Is(err, constant.NotFound) {
			// Usecase determined it's a client-side issue (NotFound)
			httpErrorHelper(span, w, r, http.StatusBadRequest, "Sorry, user not found", err)
		} else {
			// Any other error from usecase is treated as an internal server error by the handler
			httpErrorHelper(span, w, r, http.StatusInternalServerError, "Internal Server Error", err)
		}
		return
	} else {
		span.SetAttributes(attribute.String("http.res", "ok")) // Set for successful case
	}

	fmt.Fprintf(w, "Hello, %s!\n", usecaseResultName)
}

// handleQuery handles requests with query parameters like "/query?key=value"
func handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	queryValues := r.URL.Query()
	key := queryValues.Get("key")
	multiValue := queryValues["multi"]

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
