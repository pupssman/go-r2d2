# DTMF-based data transfer

## Usage

* get your byte string
* append a CRC8 checksum as a last byte
* convert to hex string
* convert each hex digit to DTMF tone code, replacing `f -> #` and `e -> *`
* generate DTMF sound sequence: 100ms per tone, no silence between tones (see `test.wav` for example)
* build and run the binary via `arecord -f FLOAT_LE -r 48000 | ./go-r2d2`, it will start waiting for input
* play generated sound, if all is OK binary will terminate and output message to STDOUT
* retry playing without restarting binary in case of failure
* run binary with `--debug 2` for maximum debug

## Encoding

Sample python encoder:

```python
import crc8

h2d = lambda x: "*" if x == "e" else ("#" if x == "f" else x)


def dtmf_encode(message):
    """
    encode given message bytestring as dtmf_codes with laste symbol of crc32 checksum
    """
    bytes_msg = map(ord, message + crc8.crc8(message).digest())
    print 'bytes are', bytes_msg
    hex_msg = "".join("{:02x}".format(a) for a in bytes_msg)
    print 'hex is', hex_msg
    return "".join(map(h2d, hex_msg)).upper()

print dtmf_encode('hello, world')
# outputs 68656C6C6#2C20776#726C6456
```

## Build

Just run `go get && go build` for local build.

Run `env GOOS=linux GOARCH=arm64 go build` to build for arm, for example.

## Test

A sample record and a script is provided. Run `go build && ./test.sh` and observe exit code.
