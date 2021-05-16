package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"os/user"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/zalando/go-keyring"
)

var buildStamp = "unreleased"

// This application exports all binance USD trades into CSV file.
// Assume interractive mode and require application to read user input prior to closing
func main() {
	err := process()
	argsWithoutProg := os.Args[1:]
	rv := 0
	if err != nil {
		fmt.Println(err.Error())
		rv = 1
	}
	// any argument would indicate non-interactive mode
	// -b is for batch or background mode (non-interactive)
	interactive := len(argsWithoutProg) == 0
	if interactive {
		fmt.Print("Press Enter to quit application:")
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
	}
	os.Exit(rv)
}

func process() error {
	var (
		apiKey    string
		secretKey string
		err       error
	)

	fmt.Println("binance-trades version: ", buildStamp)

	{
		var u *user.User
		u, err = user.Current()
		if err != nil {
			return err
		}
		fmt.Println("using username: ", u.Username)
		apiKey, err = getSecret("binance.api.key", u.Username)
		if err != nil {
			return err
		}
		secretKey, err = getSecret("binance.api.secret", u.Username)
		if err != nil {
			return err
		}
	}

	client := binance.NewClient(apiKey, secretKey)
	client.BaseURL = "https://api.binance.us"

	recvWinddow := binance.WithRecvWindow(10000)
	var account *binance.Account
	{
		svc := client.NewGetAccountService()
		account, err = svc.Do(context.Background(), recvWinddow)
		if err != nil {
			return err
		}
	}
	{
		svc := client.NewListTradesService()

		file, err := os.Create("result.csv")
		if err != nil {
			return err
		}
		defer file.Close()
		writer := csv.NewWriter(file)
		defer writer.Flush()
		writer.Write([]string{
			"Commission",
			"CommissionAsset",
			"Price",
			"Quantity",
			"QuoteQuantity",
			"Symbol",
			"ID",
			"IsBestMatch",
			"IsBuyer",
			"IsIsolated",
			"IsMaker",
			"OrderID",
			"Time",
		})

		for _, balance := range account.Balances {
			if balance.Asset == "USD" {
				continue
			}
			var free float64
			free, err = strconv.ParseFloat(balance.Free, 64)
			if err != nil {
				return err
			}
			if free == 0.0 {
				continue
			}
			fmt.Println("Asset: ", balance.Asset, ", free: ", balance.Free, ", locked: ", balance.Locked)
			svc.Symbol(balance.Asset + "USD")
			trades, err := svc.Do(context.Background(), recvWinddow)
			if err != nil {
				return err
			}
			fmt.Println("  ", balance.Asset, " trades")
			for _, trade := range trades {
				line := []string{
					trade.Commission,
					trade.CommissionAsset,
					trade.Price,
					trade.Quantity,
					trade.QuoteQuantity,
					trade.Symbol,
					fmt.Sprintf("%v", trade.ID),
					fmt.Sprintf("%v", trade.IsBestMatch),
					fmt.Sprintf("%v", trade.IsBuyer),
					fmt.Sprintf("%v", trade.IsIsolated),
					fmt.Sprintf("%v", trade.IsMaker),
					fmt.Sprintf("%v", trade.OrderID),
					fmt.Sprintf("%v", timeFromMillisec(trade.Time)),
				}
				writer.Write(line)
			}
		}
	}
	return nil
}

func timeFromMillisec(timeMillisec int64) time.Time {
	d := timeMillisec * int64(time.Millisecond)
	return time.Unix(0, d)
}

func getSecret(service, user string) (string, error) {
	var secret string
	var err error
	// get secret
	secret, err = keyring.Get(service, user)
	if err != nil {
		// assume password is not set
		secret = readValue(fmt.Sprintf("New '%s' for %s", service, user))
		// set password
		err = keyring.Set(service, user, secret)
		if err != nil {
			return "", fmt.Errorf("failed to set keyring value: %v", err)
		}
	}
	return secret, nil
}
func readValue(message string) string {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print(message)
	fmt.Print(": ")
	for scanner.Scan() {
		return scanner.Text()
	}
	return ""
}
