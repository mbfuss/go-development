package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	// Сохраняем текущее время о начале программы
	start := time.Now()

	// Определение флагов командной строки
	srcPath, dstPath := parseFlags()

	// Проверяется, возвращает ли функция validateFlags ложное значение (false)
	err := validateFlags(srcPath, dstPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Открытие файла с URL
	file, err := openFile(*srcPath)
	if err != nil {
		fmt.Printf("Ошибка при открытии файла: %s", err)
		return
	}
	// Откладывает закрытие файла до конца функции main
	// + гарантирует, что файл будет закрыт, если произойдет ошибка
	defer file.Close()

	// Создание директории назначения
	err = createDirectory(*dstPath)
	if err != nil {
		fmt.Printf("Ошибка создания директория: %s", err)
		return
	}

	// Обработка URL
	err = processURLs(file, *dstPath)
	if err != nil {
		fmt.Printf("Ошибка при обработке URL: %v", err)
		return
	}

	// Вычисление продолжительности выполнения программы
	duration := time.Since(start)
	fmt.Printf("Программа выполнилась за %v\n", duration)
}

// parseFlags - парсинг флагов командной строки.
func parseFlags() (srcPath *string, dstPath *string) {
	srcPath = flag.String("src", "", "Путь файла со списком URL")
	dstPath = flag.String("dst", "", "Путь для спаршенных файлов")
	flag.Parse()
	return srcPath, dstPath
}

// validateFlags - проверка значений флагов
func validateFlags(srcPath, dstPath *string) error {
	if *srcPath == "" || *dstPath == "" {
		return fmt.Errorf("Используйте: ./grabber --src=source.txt --dst=destination")
	}
	return nil
}

// openFile - открытие файла с URL
func openFile(path string) (*os.File, error) {
	return os.Open(path)
}

// createDirectory - создание директории назначения
func createDirectory(path string) error {
	// os.MkdirAll: Функция которая рекурсивно создает все указанные в пути директории
	// *dstPath: Разыменование указателя на строку, которая содержит путь к директории назначения, указанный пользователем в флаге --dst
	// MkdirAll(path string, perm os.FileMode) error
	return os.MkdirAll(path, os.ModePerm)
}

// processURLs - обработка URL из файла
func processURLs(file *os.File, dstPath string) error {
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		url := scanner.Text()
		if err := treatmentURL(url, dstPath); err != nil {
			fmt.Printf("Ошибка при чтении URL %s: %v\n", url, err)
		}
	}
	return scanner.Err()
}

// treatmentURL - функция обработки URL
func treatmentURL(url string, dstPath string) error {
	fmt.Printf("Обработка URL: %s", url)

	// Выполняет HTTP GET запрос к указанному URL
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("Ошибка при подключении к URL %s", err)
	}
	// Откладывает закрытие тела ответа до конца функции. Это гарантирует, что ресурс будет освобожден
	defer resp.Body.Close()

	// Проверяет статус ответа. Если он не равен 200 OK, возвращает ошибку
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Non-OK HTTP статус: %s", resp.Status)
	}

	// Определяет имя файла для сохранения содержимого, используя путь к директории назначения и безопасное имя файла
	filename := filepath.Join(dstPath, sanitizeFilename(url)+".html")

	// Создает файл для записи содержимого
	outFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("Ошибка при создании файла для записи: %s", err)
	}
	// Откладывает закрытие файла до конца функции
	defer outFile.Close()

	// Считывает содержимое тела ответа и записывает его в открытый файл
	_, err = outFile.ReadFrom(resp.Body)
	if err != nil {
		return fmt.Errorf("Ошибка при записи данных в файл: %s", err)
	}

	fmt.Printf("Сохранение %s в %s\n", url, filename)

	// Возвращает nil, если ошибок нет
	return nil
}

// sanitizeFilename - принимает строку (URL) и заменяет все символы / на символ _
func sanitizeFilename(url string) string {
	return strings.ReplaceAll(url, "/", "_")
}
