package incarne

import (
	"regexp"
	"strconv"
)

type ExtractRegex struct {
	Re      *regexp.Regexp
	GroupNo uint // group 0 means whole match that were found
	MatchNo int  // -1 means random
}

func (r *ExtractRegex) String() string {
	return r.Re.String() + " group " + strconv.Itoa(int(r.GroupNo)) + " match " + strconv.Itoa(r.MatchNo)
}

type InputItem struct {
	Hostname string
	Payload  []byte
	RegexOut map[string]*ExtractRegex
}

func (i *InputItem) ReplaceValues(values map[string][]byte) {
	for name, val := range values {
		re := regexp.MustCompile("\\$\\{" + name + "}")
		i.Payload = re.ReplaceAll(i.Payload, val)
	}
}
