package main

import (
	"fmt"
	"os"

	c "github.com/ipfs/go-cid"

	mh "github.com/multiformats/go-multihash"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s prefix ...\n", os.Args[0])
		os.Exit(1)
	}
	switch os.Args[1] {
	case "prefix":
		err := prefixCmd(os.Args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "usage: %s prefix ...\n")
		os.Exit(1)
	}
}

func prefixCmd(args []string) error {
	for _, cid := range args {
		p, err := prefix(cid)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "%s\n", p)
	}
	return nil
}

func prefix(str string) (string, error) {
	cid, err := c.Decode(str)
	if err != nil {
		return "", err
	}
	p := cid.Prefix()
	return fmt.Sprintf("cidv%d-%s-%s-%d",
		p.Version,
		codecToStr(p.Codec),
		mhToStr(p.MhType),
		p.MhLength,
	), nil
}

func codecToStr(num uint64) string {
	name, ok := c.CodecToStr[num]
	if !ok {
		return fmt.Sprintf("c?%d", num)
	}
	return name
}

func mhToStr(num uint64) string {
	name, ok := mh.Codes[num]
	if !ok {
		return fmt.Sprintf("h?%d", num)
	}
	return name
}
