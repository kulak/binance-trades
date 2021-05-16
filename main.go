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

// This application exports all binance USD trades into CSV file.
func main() {
	var (
		apiKey    string
		secretKey string
		err       error
	)

	{
		var u *user.User
		u, err = user.Current()
		if err != nil {
			panic(err)
		}
		fmt.Println("using username: ", u.Username)
		apiKey, err = getSecret("binance.api.key", u.Username)
		if err != nil {
			panic(err)
		}
		secretKey, err = getSecret("binance.api.secret", u.Username)
		if err != nil {
			panic(err)
		}
	}

	client := binance.NewClient(apiKey, secretKey)
	client.TimeOffset = 5000
	client.BaseURL = "https://api.binance.us"

	{
		start := time.Now()
		timeMillisec, err := client.NewServerTimeService().Do(context.Background())
		end := time.Now()
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}
		serverTime := timeFromMillisec(timeMillisec)
		fmt.Printf("Start time:  %v\n", start)
		fmt.Printf("End time:    %v, duration: %v\n", end, end.Sub(start))
		fmt.Printf("Server time: %v\n", serverTime)
	}

	recvWinddow := binance.WithRecvWindow(10000)
	var account *binance.Account
	{
		svc := client.NewGetAccountService()
		account, err = svc.Do(context.Background(), recvWinddow)
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}
		fmt.Println("account:")
		fmt.Println(account)
	}
	{
		svc := client.NewListTradesService()

		file, err := os.Create("result.csv")
		if err != nil {
			panic(err)
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
				panic(err)
			}
			if free == 0.0 {
				continue
			}
			fmt.Println("Asset: ", balance.Asset, ", free: ", balance.Free, ", locked: ", balance.Locked)
			svc.Symbol(balance.Asset + "USD")
			trades, err := svc.Do(context.Background(), recvWinddow)
			if err != nil {
				fmt.Println("Error: ", err)
				return
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
