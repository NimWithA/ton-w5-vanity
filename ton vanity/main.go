package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xssnick/tonutils-go/ton/wallet"
)

const Version = "1.0.0"

type Match struct {
	Address  string
	Mnemonic string
}

var (
	printMu  sync.Mutex
	lastRate uint64
)

func readLine(reader *bufio.Reader) (string, error) {
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

func askString(reader *bufio.Reader, question string, required bool, def string) string {
	for {
		if def != "" {
			fmt.Printf("%s [%s]: ", question, def)
		} else {
			fmt.Printf("%s: ", question)
		}

		input, err := readLine(reader)
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		if input == "" {
			if def != "" {
				return def
			}
			if required {
				fmt.Println("This value is required. Please enter something.")
				continue
			}
			return ""
		}

		return input
	}
}

func askInt(reader *bufio.Reader, question string, def int) int {
	for {
		if def != 0 {
			fmt.Printf("%s [%d]: ", question, def)
		} else {
			fmt.Printf("%s [0]: ", question)
		}

		input, err := readLine(reader)
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		if input == "" {
			return def
		}

		v, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("Please enter a valid integer number.")
			continue
		}

		return v
	}
}

func askBool(reader *bufio.Reader, question string, def bool) bool {
	defStr := "y"
	if !def {
		defStr = "n"
	}

	for {
		fmt.Printf("%s [y/n, default %s]: ", question, defStr)
		input, err := readLine(reader)
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		if input == "" {
			return def
		}

		input = strings.ToLower(input)
		if input == "y" || input == "yes" {
			return true
		}
		if input == "n" || input == "no" {
			return false
		}

		fmt.Println("Please answer with 'y' or 'n'.")
	}
}

func highlightPattern(addr string, pattern string, patternUpper string, caseSensitive bool) string {
	if pattern == "" {
		return addr
	}

	if caseSensitive {
		idx := strings.Index(addr, pattern)
		if idx < 0 {
			return addr
		}
		return addr[:idx] + "\033[32m" + addr[idx:idx+len(pattern)] + "\033[0m" + addr[idx+len(pattern):]
	}

	upperAddr := strings.ToUpper(addr)
	idx := strings.Index(upperAddr, patternUpper)
	if idx < 0 {
		return addr
	}
	return addr[:idx] + "\033[32m" + addr[idx:idx+len(pattern)] + "\033[0m" + addr[idx+len(pattern):]
}

func printStats(total, rate uint64) {
	printMu.Lock()
	defer printMu.Unlock()
	fmt.Printf("\r\033[2K\033[36m[stats]\033[0m generated=%d, rate≈%d wallets/sec", total, rate)
}

func printHit(hits int, addr string, pattern string, patternUpper string, caseSensitive bool) {
	printMu.Lock()
	defer printMu.Unlock()

	highlighted := highlightPattern(addr, pattern, patternUpper, caseSensitive)

	fmt.Printf("\r\033[2K\033[32m[hit %d]\033[0m %s\n", hits, highlighted)
}

func worker(
	ctx context.Context,
	id int,
	cfg wallet.ConfigV5R1Final,
	pattern string,
	patternUpper string,
	caseSensitive bool,
	matchCh chan<- Match,
	totalGenerated *uint64,
	debug bool,
) {
	defer func() {
		if debug {
			log.Printf("[worker %d] stopped", id)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		words := wallet.NewSeed()

		w, err := wallet.FromSeedWithOptions(
			nil,
			words,
			cfg,
		)
		if err != nil {
			if debug {
				log.Printf("[worker %d] FromSeedWithOptions error: %v", id, err)
			}
			continue
		}

		addr := w.Address()
		if addr == nil {
			if debug {
				log.Printf("[worker %d] nil address returned", id)
			}
			continue
		}

		addrCopy := addr.Copy()
		addrCopy.SetBounce(false)
		addrStr := addrCopy.String()

		atomic.AddUint64(totalGenerated, 1)

		var match bool
		if caseSensitive {
			match = strings.Contains(addrStr, pattern)
		} else {
			match = strings.Contains(strings.ToUpper(addrStr), patternUpper)
		}

		if match {
			m := Match{
				Address:  addrStr,
				Mnemonic: strings.Join(words, " "),
			}

			select {
			case matchCh <- m:
				if debug {
					log.Printf("[worker %d] MATCH: %s", id, addrStr)
				}
			case <-ctx.Done():
				return
			}
		}
	}
}

func main() {
	preventSleep()
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\033[36m=======================================================\033[0m")
	fmt.Println("              \033[1mTON W5 Vanity Address Generator\033[0m")
	fmt.Println("                   \033[32mDeveloped by NimWithA\033[0m")
	fmt.Println("\033[36m=======================================================\033[0m")
	fmt.Println()
	fmt.Println("\033[33m⚠  Important Notices:\033[0m")
	fmt.Println("\033[33m• This tool generates TON W5 (V5R1) wallets locally.\033[0m")
	fmt.Println("\033[33m• No internet connection is required — fully offline.\033[0m")
	fmt.Println("\033[33m• CPU usage may reach 100% depending on worker count.\033[0m")
	fmt.Println("\033[33m• Store generated mnemonics securely. Do NOT share them.\033[0m")
	fmt.Println()
	fmt.Printf("\033[90mVersion %s\033[0m\n", Version)
	fmt.Println()

	pattern := askString(reader,
		"1) Enter the text pattern to search inside the address (required)",
		true,
		"",
	)

	caseSensitive := askBool(reader,
		"2) Should the pattern be case-sensitive?",
		false,
	)

	defaultWorkers := runtime.NumCPU()
	workers := askInt(reader,
		fmt.Sprintf("3) Number of workers (parallel generators). Recommended: number of CPU cores"),
		defaultWorkers,
	)

	maxHits := askInt(reader,
		"4) Maximum number of matches (0 = unlimited until you stop the program)",
		0,
	)

	network := askString(reader,
		"5) Network (mainnet/testnet)",
		false,
		"mainnet",
	)
	network = strings.ToLower(network)
	if network != "mainnet" && network != "testnet" {
		fmt.Println("Unknown network, using 'mainnet' as default.")
		network = "mainnet"
	}

	outFile := askString(reader,
		"6) Output file name for saving matches",
		false,
		"matches.txt",
	)

	debug := askBool(reader,
		"7) Enable debug logs",
		false,
	)

	showStats := askBool(reader,
		"8) Show performance statistics (wallets per second)",
		true,
	)

	fmt.Println()
	fmt.Println("-------------------------------------------------------")
	fmt.Println("Configuration summary:")
	fmt.Printf("  Pattern           : %s\n", pattern)
	fmt.Printf("  Case-sensitive    : %t\n", caseSensitive)
	fmt.Printf("  Workers           : %d\n", workers)
	if maxHits == 0 {
		fmt.Printf("  Max matches       : unlimited\n")
	} else {
		fmt.Printf("  Max matches       : %d\n", maxHits)
	}
	fmt.Printf("  Network           : %s\n", network)
	fmt.Printf("  Output file       : %s\n", outFile)
	fmt.Printf("  Debug logs        : %t\n", debug)
	fmt.Printf("  Show stats        : %t\n", showStats)
	fmt.Println("-------------------------------------------------------")

	startNow := askBool(reader, "Start generation with these settings?", true)
	if !startNow {
		fmt.Println("Aborted by user. Exiting.")
		return
	}

	if workers <= 0 {
		workers = 1
	}

	cfg := wallet.ConfigV5R1Final{
		NetworkGlobalID: wallet.MainnetGlobalID,
		Workchain:       0,
	}

	if network == "testnet" {
		cfg.NetworkGlobalID = wallet.TestnetGlobalID
	}

	if debug {
		log.Printf("Starting vanity generator | pattern=%q | caseSensitive=%t | workers=%d | network=%s | out=%s | max=%d",
			pattern, caseSensitive, workers, network, outFile, maxHits)
	}

	patternUpper := strings.ToUpper(pattern)

	f, err := os.OpenFile(outFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		log.Fatalf("Cannot open output file %s: %v", outFile, err)
	}
	defer f.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	matchCh := make(chan Match, workers*4)

	var wg sync.WaitGroup
	var totalGenerated uint64

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			worker(ctx, id, cfg, pattern, patternUpper, caseSensitive, matchCh, &totalGenerated, debug)
		}(i)
	}

	go func() {
		wg.Wait()
		close(matchCh)
	}()

	if showStats {
		go func() {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()

			var lastCount uint64
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					current := atomic.LoadUint64(&totalGenerated)
					diff := current - lastCount
					lastCount = current
					rate := diff / 2
					atomic.StoreUint64(&lastRate, rate)
					printStats(current, rate)
				}
			}
		}()
	}

	fmt.Println()
	fmt.Println("Generation started. Press Ctrl+C to stop at any time.")
	fmt.Println()

	hits := 0
	for m := range matchCh {
		hits++

		if _, err := f.WriteString(
			fmt.Sprintf("ADDRESS: %s\nMNEMONIC: %s\n---\n", m.Address, m.Mnemonic),
		); err != nil {
			log.Printf("Error writing to file: %v", err)
		}

		printHit(hits, m.Address, pattern, patternUpper, caseSensitive)

		if showStats {
			total := atomic.LoadUint64(&totalGenerated)
			rate := atomic.LoadUint64(&lastRate)
			printStats(total, rate)
		}

		if maxHits > 0 && hits >= maxHits {
			if debug {
				log.Printf("Reached maximum matches (%d). Stopping...", maxHits)
			}
			cancel()
			break
		}
	}

	printMu.Lock()
	fmt.Print("\n")
	printMu.Unlock()

	fmt.Fprintf(os.Stderr, "Done. Total wallets generated: %d, matches found: %d\n",
		atomic.LoadUint64(&totalGenerated), hits)

	fmt.Println()
	fmt.Println("Generation finished. You can now safely close this window.")
	fmt.Println("Press ENTER to exit...")
	_, _ = reader.ReadString('\n')
}
