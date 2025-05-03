package handler

import "net/http"

// Создаем обертку для ResponseWriter, чтобы отслеживать статус-код
type statusCodeTracker struct {
	http.ResponseWriter
	statusCode int
}

// Переопределяем WriteHeader для отслеживания статус-кода
func (w *statusCodeTracker) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// Write также может вызывать WriteHeader неявно
func (w *statusCodeTracker) Write(b []byte) (int, error) {
	// Если статус-код еще не был установлен, устанавливаем 200 OK по умолчанию
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}
