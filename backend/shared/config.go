package shared

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	ZINC_HOST     string
	ZINC_INDEX    string
	ZINC_USER     string
	ZINC_PASSWORD string
}

func loadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening .env file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		value = strings.Trim(value, `"'`)

		err := os.Setenv(key, value)
		if err != nil {
			return fmt.Errorf("error setting environment variable %s: %v", key, err)
		}
	}
	return scanner.Err()
}

func LoadConfig() (*Config, error) {
	err := loadEnvFile(".env")
	if err != nil {
		return nil, fmt.Errorf("error loading .env file: %v", err)
	}

	config := &Config{}

	config.ZINC_HOST = os.Getenv("ZINC_HOST")
	config.ZINC_USER = os.Getenv("ZINC_USER")
	config.ZINC_PASSWORD = os.Getenv("ZINC_PASSWORD")
	config.ZINC_INDEX = os.Getenv("ZINC_INDEX")

	if config.ZINC_HOST == "" || config.ZINC_USER == "" ||
		config.ZINC_PASSWORD == "" || config.ZINC_INDEX == "" {
		return nil, fmt.Errorf("required environment variables are missing")
	}

	return config, nil
}
