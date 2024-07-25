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

	// Позволяет основному потоку (или главной горутине) блокироваться до тех пор,
	// пока все запущенные горутины не завершат свою работу
	var wg sync.WaitGroup
	// Используется для обеспечения безопасного доступа к разделяемым данным из нескольких горутин
	// В данном случае она используется для синхронизации доступа к срезу ошибок errors,
	// чтобы избежать гонок данных при записи ошибок из различных горутин
	var mu sync.Mutex
	errors := []error{}

	// Чтение URL из файла(построчно) и запуск горутин для их обработки
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// Возвращает строку, считанную в текущий момент из файла
		url := scanner.Text()
		wg.Add(1)
		go func(url string) {
			// Уменьшение счетчика, по окончании выполнения функции
			defer wg.Done()
			err := treatmentURL(url, *dstPath)
			if err != nil {
				// Строка кода блокирует мьютекс
				// Если другой поток уже заблокировал мьютекс, текущая горутина будет ждать, пока мьютекс не станет доступным
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			}
			// Cоздает копию url для каждой горутины, что гарантирует,
			// что каждая горутина работает с тем значением url, которое было на момент ее создания
		}(url)
	}

	// Ждем завершения всех горутин
	wg.Wait()

	// Вывод всех ошибок, если они есть
	for _, err := range errors {
		fmt.Printf("Ошибка: %v\n", err)
	}

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
