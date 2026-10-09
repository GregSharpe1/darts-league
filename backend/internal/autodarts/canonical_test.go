package autodarts

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
)

func TestCanonicalMatchesNode(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node unavailable")
	}
	// Numbers include exponent thresholds, subnormal, rounding, and negative zero;
	// keys include astral/BMP ordering, which differs from Go UTF-8 string order.
	body := `{"\ue000":1,"\ud83d\ude00":2,"numbers":[-0,1e-7,1e-6,1e20,1e21,5e-324,333333333.33333329,1.7976931348623157e308],"text":"<>&\u2028\u2029"}`
	cmd := exec.Command("node", "-e", `let s='';process.stdin.on('data',b=>s+=b);process.stdin.on('end',()=>{function c(v){if(Array.isArray(v))return '['+v.map(c).join(',')+']';if(v!==null&&typeof v==='object')return '{'+Object.keys(v).sort().map(k=>JSON.stringify(k)+':'+c(v[k])).join(',')+'}';return JSON.stringify(v)}process.stdout.write(c(JSON.parse(s)))})`)
	cmd.Stdin = strings.NewReader(body)
	want, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	got, err := jsoncanonicalizer.Transform([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("canonical mismatch\nGo: %s\nNode: %s", got, want)
	}
	legacy, err := Parse(example(t, "legacy"))
	if err != nil {
		t.Fatal(err)
	}
	if legacy.Digest != "6b3471c50140c311a64d35f4b4a88cf6751e625513222bfd8aab33f714ef447d" {
		t.Fatal("contract digest mismatch")
	}
}

func TestRejectsUnicodeAndResourceAbuse(t *testing.T) {
	base := string(example(t, "missing_detail"))
	for _, extra := range []string{`"x":"\ud800",`, `"x":1e999,`, `"x":` + strings.Repeat("[", 12) + "0" + strings.Repeat("]", 12) + ",", `"x":{"a":1,"\u0061":2},`} {
		if _, err := Parse([]byte(strings.Replace(base, "{", "{"+extra, 1))); err == nil {
			t.Fatalf("accepted %s", extra)
		}
	}
	if _, err := Parse(append([]byte{0xff}, []byte(base)...)); err == nil {
		t.Fatal("accepted invalid UTF8")
	}
}
