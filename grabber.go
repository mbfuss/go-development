package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func main() {
	// Сохраняем текущее время начала программы для последующего вычисления времени выполнения
	start := time.Now()

	// Определяем флаги командной строки для входного и выходного путей
	srcPath, dstPath := parseFlags()

	// Проверяем, что флаги src и dst были заданы корректно
	err := validateFlags(srcPath, dstPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Открываем файл, содержащий список URL
	file, err := openFile(*srcPath)
	if err != nil {
		fmt.Printf("Ошибка при открытии файла: %s\n", err)
		return
	}
	// Гарантируем закрытие файла по завершении функции main
	defer file.Close()

	// Создаем директорию назначения для сохранения загруженных файлов
	err = createDirectory(*dstPath)
	if err != nil {
		fmt.Printf("Ошибка создания директории: %s\n", err)
		return
	}

	// Создаем канал для передачи URL
	urls := make(chan string)
	// Создаем канал для передачи ошибок
	errors := make(chan error)
	// Создаем канал для сигнала завершения работы горутин
	done := make(chan struct{})

	var wg sync.WaitGroup

	// Запускаем несколько горутин для параллельной обработки URL
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go worker(urls, *dstPath, errors, &wg)
	}

	// Запускаем горутину для чтения и вывода ошибок
	// анонимная функция
	go func() {
		for err := range errors {
			fmt.Printf("Ошибка: %v\n", err)
		}
	}()

	// Запускаем горутину для чтения URL из файла и отправки их в канал
	go func() {
		err := processURLs(file, urls)
		if err != nil {
			fmt.Printf("Ошибка при обработке URL: %v\n", err)
		}
		// Закрываем канал URL после завершения чтения
		close(urls)
	}()

	// Ждем завершения всех горутин
	wg.Wait()
	// Закрываем канал done, чтобы сигнализировать об окончании работы
	close(done)

	// Вычисляем и выводим продолжительность выполнения программы
	duration := time.Since(start)
	fmt.Printf("Программа выполнилась за %v\n", duration)
}

// parseFlags - функция для создания флагов командной строки
func parseFlags() (srcPath *string, dstPath *string) {
	srcPath = flag.String("src", "", "Путь к файлу со списком URL")
	dstPath = flag.String("dst", "", "Путь к директории для сохранения загруженных файлов")
	flag.Parse()
	return srcPath, dstPath
}

// validateFlags - функция для проверки значений флагов
func validateFlags(srcPath, dstPath *string) error {
	if *srcPath == "" || *dstPath == "" {
		return fmt.Errorf("Используйте: ./grabber --src=source.txt --dst=destination")
	}
	return nil
}

// openFile - функция для открытия файла с URL
func openFile(path string) (*os.File, error) {
	return os.Open(path)
}

// createDirectory - функция для создания директории назначения
func createDirectory(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

// processURLs - функция для чтения URL из файла и отправки их в канал
func processURLs(file *os.File, urls chan<- string) error {
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		url := scanner.Text()
		// Отправляем URL в канал
		urls <- url
	}
	return scanner.Err()
}

// worker - горутина для обработки URL
func worker(urls <-chan string, dstPath string, errors chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()
	for url := range urls {
		if err := treatmentURL(url, dstPath); err != nil {
			errors <- err
		}
	}
}

// treatmentURL - функция для обработки каждого URL
func treatmentURL(url string, dstPath string) error {
	fmt.Printf("Обработка URL: %s\n", url)

	// Выполняем HTTP GET запрос к указанному URL
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("Ошибка при подключении к URL %s: %v", url, err)
	}
	// Гарантируем закрытие тела ответа после завершения функции
	defer resp.Body.Close()

	// Проверяем статус ответа. Если он не равен 200 OK, возвращаем ошибку
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Non-OK HTTP статус: %s", resp.Status)
	}

	// Определяем имя файла для сохранения содержимого, используя безопасное имя файла
	filename := filepath.Join(dstPath, sanitizeFilename(url)+".html")

	// Создаем файл для записи содержимого
	outFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("Ошибка при создании файла для записи: %v", err)
	}
	// Гарантируем закрытие файла после завершения функции
	defer outFile.Close()

	// Считываем содержимое тела ответа и записываем его в открытый файл
	_, err = outFile.ReadFrom(resp.Body)
	if err != nil {
		return fmt.Errorf("Ошибка при записи данных в файл: %v", err)
	}

	fmt.Printf("Сохранение %s в %s\n", url, filename)
	return nil
}

// sanitizeFilename - функция для замены всех символов / на _ в URL, чтобы создать безопасное имя файла
func sanitizeFilename(url string) string {
	return strings.ReplaceAll(url, "/", "_")
}
