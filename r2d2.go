package main

import (
    "os"
    "fmt"
    "log"
    "flag"
    "encoding/hex"

    "github.com/mjibson/go-dsp/wav"
    "gonum.org/v1/gonum/floats"
    "github.com/sigurn/crc8"
)

var DEBUG = flag.Int("debug", 0, "Debug level -- 0 for none, 1 for info, 2 for detailed")

func log_info(v ...interface{}) {
    if *DEBUG > 0 {
        log.Println(v)
    }
}

func log_debug(v ...interface{}) {
    if *DEBUG > 1 {
        log.Println(v)
    }
}

func log_error(v ...interface{}) {
    if *DEBUG >= 0 {
        log.Println("ERROR:", v)
    }
}

// DTMF code table, see https://en.wikipedia.org/wiki/DTMF
var DTMF_FIRST = [...]int {697, 770, 852, 941}
var DTMF_SECOND = [...]int {1209, 1336, 1477, 1633}

var DTMF_MAP = map[int]map[int]string{
    697: map[int]string{1209: "1", 1336: "2", 1477: "3", 1633: "A"},
    770: map[int]string{1209: "4", 1336: "5", 1477: "6", 1633: "B"},
    852: map[int]string{1209: "7", 1336: "8", 1477: "9", 1633: "C"},
    941: map[int]string{1209: "*", 1336: "0", 1477: "#", 1633: "D"},
}

// convert dtmf hexy string to proper hex string
func dtmf_to_hex(d string) (string) {
    var res = ""
    for _, a := range d {
        if a == '*' {
            res += "e"
        } else if a == '#' {
            res += "f"
        } else {
            // assume no other letters possible
            res += string(a)
        }

    }
    return res
}

// reads dtmf codes from stdin and sends them down the given channel as DTMF_MAP value strings
func dtmf_reader(c chan string) {
    var w, err = wav.New(os.Stdin)

    if err != nil {
        log_error(err)
        return
    }

    var sr = w.Header.SampleRate

    frequencies_low := []float32 {697, 770, 852, 941}
    frequencies_high := []float32 {1209, 1336, 1477, 1633}

    var goert_low = InitGoertzel(int(sr), 1024 * 1, frequencies_low)
    var goert_high = InitGoertzel(int(sr), 1024 * 1, frequencies_high)

    var lows = make([]float64, len(frequencies_low))
    var highs = make([]float64, len(frequencies_high))

    var LOW = 1.0
    var HIGH = 0.1
    var DETECT = float64(2.5)

    var TONE_DURATION = 0.10  // tone in a signal should sound for exactly that
    var SPLITS = uint32(4) // split each tone duration to 4 intervals so we always have 3 intervals inside of a tone
    var batch = int(float64(sr / SPLITS) * TONE_DURATION)

    // TODO: be sure about the sample rate!
    log_info("Sample rate", sr, "batch read", batch)

    for {
        var samples, err = w.ReadFloats(batch)

        if err != nil {
            log_error(err)
            continue
        }

        goert_low.ResetGoertzel()
        goert_high.ResetGoertzel()

        for _, sample := range samples {
            goert_low.ProcessSample(sample)
            goert_high.ProcessSample(sample)
        }

        goert_low.calcMagnitudeSquared()
        goert_high.calcMagnitudeSquared()

        for n, fb := range goert_low.freqBucket {
            lows[n] = float64(fb.MagnitureSquared)
        }

        for n, fb := range goert_high.freqBucket {
            highs[n] = float64(fb.MagnitureSquared)
        }

        var low_a = floats.Sum(lows) / float64(len(lows))
        var high_a = floats.Sum(highs) / float64(len(highs))

        var low int
        var high int

        for n, l := range lows {
            if l > LOW && l > low_a * DETECT {
                low = int(frequencies_low[n])
            }
        }

        for m, h := range highs {
            if h > HIGH && h > high_a * DETECT {
                high = int(frequencies_high[m])
                log_debug("Choosing high", h, HIGH, high_a, DETECT, m)
            }
        }

        var res string

        if low > 0 && high > 0 {
            res = DTMF_MAP[low][high]
        } else {
            res = ""
        }

        log_debug(lows, highs, low_a, high_a, low_a * DETECT, high_a * DETECT, low, high, "res: <", res, ">")

        c <- res
    }
}

func todoubles(in []float32) ([]float64) {
    var result = make([]float64, len(in))

    for i := range in {
        result[i] = float64(in[i])
    }

    return result
}

// detects full characters from the sequence
// sequence should have at least three (but maybe four) consequitive symbols for each frame
func deduplicate(in chan string, out chan string) {
    var val = "XXX"
    var count = 0

    for {
        code := <-in

        log_info("raw msg <", code, ">, current", val, "with count", count)

        if count == 3 {
            out <- val
            if val == code {
                // got 4th byte of frame, send frame
                count = 0
                val = code
            } else {
                // got part of next frame, send prev
                count = 1
                val = code
            }
        } else {
            if val == code {
                count += 1
            } else {
                val = code
                count = 1
            }
        }

    }
}

// extract message from incoming deduplicated stream
// consider everything being not blank a part of message, expect long silence at start / end of msg
func parse_message(in chan string, out chan string) {
    var blanks = 0
    var msg = ""

    for {
        m := <-in

        log_info("Parsing char<", m, ">, blanks", blanks, ", message", msg)

        if m  == "" {
            blanks += 1

            if blanks >= 5  && len(msg) > 0 {
                out <- msg
                blanks = 0
                msg = ""
            }
        } else {
            msg += m
        }
    }
}

// for each message received decode if from dtmf single-digit hexy string
// and check crc8 checksum in the last byte
// if all is ok pass decoded text down the out (stripped from checksum)
func decode_and_verify_crc8(in chan string, out chan string) {
    table := crc8.MakeTable(crc8.CRC8)

    for {
        msg := <-in
        log_info("Got message", msg)

        decoded, err := hex.DecodeString(dtmf_to_hex(msg))

        if err != nil {
            log_error(err)
            continue
        }

        log_info("Decoded to", decoded)

        crc8_calc := crc8.Checksum(decoded[:len(decoded) - 1], table)  // calc crc8 of all but the last byte of msg
        crc8_read := decoded[len(decoded) - 1] // last byte should be the crc8 checksum

        log_info("Checksum calculated", crc8_calc, "read", crc8_read)

        if crc8_calc == crc8_read {
            out <- string(decoded[:len(decoded) - 1])
        }
    }
}

// wraps given in-channel via func fn that should take first arg of in-channel and second-arg of out-channel
func chains(fn func(chan string, chan string), in chan string) chan string {
    out := make(chan string)
    go fn(in, out)
    return out
}

func main() {
    flag.Parse()
    log.SetOutput(os.Stderr)

    var raw_codes = make(chan string)
    go dtmf_reader(raw_codes)

    var dedup_codes = chains(deduplicate, raw_codes)
    var dtmf_messages = chains(parse_message, dedup_codes)
    var messages = chains(decode_and_verify_crc8, dtmf_messages)

    message := <-messages
    log_info("Read full message", message)

    fmt.Println(message)
}
