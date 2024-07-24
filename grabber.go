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
	// 	This declares an integer flag, -n, stored in the pointer nFlag, with type *int:
	// var nFlag = flag.Int("n", 1234, "help message for flag n")
	// "" - значение по умолчанию
	srcPath := flag.String("src", "", "Путь файла со списком URL")
	dstPath := flag.String("dst", "", "Путь для спаршенных файлов")
	// Создание флагов
	flag.Parse()

	// Проверяет, были ли заданы значения∏ флагов --src и --dst, если нет, то print и завершение программы
	if *srcPath == "" || *dstPath == "" {
		fmt.Println("Используйте: ./crabber --src=source.txt --dst=destination ")
		return
	}

	// Открытие файла с URL, путь на файл указывается при вводе команды ./crabber...
	file, err := os.Open(*srcPath)
	if err != nil {
		// В данном случае %v будет заменен строковым представлением ошибки, хранящейся в переменной err
		fmt.Printf("Ошибка при открытии файла: %s", err)
		return
	}
	// Откладывает закрытие файла до конца функции main
	// + гарантирует, что файл будет закрыт, если произойдет ошибка
	defer file.Close()

	// Создание директории назначения, если ее нет
	// os.MkdirAll: Функция которая рекурсивно создает все указанные в пути директории
	// *dstPath: Разыменование указателя на строку, которая содержит путь к директории назначения, указанный пользователем в флаге --dst
	// os.ModePerm: Константа из пакета os, которая задает права доступа к создаваемым директориям. Все пользователи могут читать, писать и выполнять файлы
	// MkdirAll(path string, perm os.FileMode) error
	err = os.MkdirAll(*dstPath, os.ModePerm)
	if err != nil {
		fmt.Printf("Ошибка создания директория: %s", err)
		return
	}

	// Создает новый сканер для чтения файла
	// Сканы используются для построчного чтения данных из файла
	scanner := bufio.NewScanner(file)
	// Продолжает выполняться, пока в файле есть строки для чтения
	for scanner.Scan() {
		// Копирование строки
		url := scanner.Text()
		err := treatmentURL(url, *dstPath)
		if err != nil {
			fmt.Printf("Ошибка при чтении URL %s: %v\n", url, err)
		}
	}

	// Вычисление продолжительности выполнения программы
	duration := time.Since(start)
	fmt.Printf("Программа выпонилась за %v\n", duration)
}

// Функция обработки URL
// принимает URL и путь к директории назначения, возвращает ошибку
func treatmentURL(url string, dstPath string) error {
	fmt.Printf("Обработка URL: %s", url)

	// Выполняет HTTP GET запрос к указанному URL
	resp, err := http.Get(url)
	if err != nil {
		fmt.Errorf("Ошибка при подключении к URL %s", err)
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

	fmt.Printf("Saved %s to %s\n", url, filename)

	// Возвращает nil, если ошибок нет
	return nil

}

// Принимает строку (URL) и заменяет все символы / на символ _
// Трбуется для корректного отображения создаваемого файла в файловой системе
func sanitizeFilename(url string) string {
	return strings.ReplaceAll(url, "/", "_")
}
