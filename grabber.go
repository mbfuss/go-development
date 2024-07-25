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

	// Создаем и запускаем обработку URL
	errors := processURLs(file, *dstPath)

	// Вывод всех ошибок, если они есть
	for _, err := range errors {
		fmt.Printf("Ошибка: %v\n", err)
	}

	// Вычисляем и выводим продолжительность выполнения программы
	duration := time.Since(start)
	fmt.Printf("Программа выполнилась за %v\n", duration)
}

// processURLs - функция для обработки URL из файла
func processURLs(file *os.File, dstPath string) []error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		url := scanner.Text()
		wg.Add(1)
		go func(url string, wg *sync.WaitGroup, dstPath string) {
			defer wg.Done()
			err := treatmentURL(url, dstPath)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			}
		}(url, &wg, dstPath)
	}

	wg.Wait()
	return errors
}

// parseFlags - функция для создания флагов командной строки
func parseFlags() (srcPath *string, dstPath *string) {
	srcPath = flag.String("src", "", "Путь к файлу со списком URL")
	dstPath = flag.String("dst", "", "Путь к директории для сохранения загруженных файлов")
	flag.Parse()

	if *srcPath == "" || *dstPath == "" {
		flag.Usage()
	}

	return srcPath, dstPath
}

// openFile - функция для открытия файла с URL
func openFile(path string) (*os.File, error) {
	return os.Open(path)
}

// createDirectory - функция для создания директории назначения
func createDirectory(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

// treatmentURL - функция для обработки каждого URL
func treatmentURL(url string, dstPath string) error {
	fmt.Printf("Обработка URL: %s\n", url)

	// Выполняем HTTP GET запрос к указанному URL
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("ошибка при подключении к URL %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Non-OK HTTP статус: %s", resp.Status)
	}

	filename := filepath.Join(dstPath, sanitizeFilename(url)+".html")
	outFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("ошибка при создании файла для записи: %v", err)
	}
	defer outFile.Close()

	_, err = outFile.ReadFrom(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка при записи данных в файл: %v", err)
	}

	fmt.Printf("Сохранение %s в %s\n", url, filename)
	return nil
}

// sanitizeFilename - функция для замены всех символов / на _ в URL, чтобы создать безопасное имя файла
func sanitizeFilename(url string) string {
	return strings.ReplaceAll(url, "/", "_")
}
