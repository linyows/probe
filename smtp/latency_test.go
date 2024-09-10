package smtp

import (
	"strings"
	"testing"
)

func TestGetLatency(t *testing.T) {
	data, err := getLatency("./testdata/", true)
	if err != nil {
		t.Errorf("got error %s", err)
	}
	expects := `Time (s), Received Time(s), Latency (s), Sub Latency (s), Sent at, First Received at, Received at, File, Return-Path
0, 0, 0, 0, 2024-08-25 16:55:51, 2024-08-25 16:55:51, 2024-08-25 16:55:51, 1724572551.V802I9c075dM312459.mx1, alice@msa1.local
220, 2658, 2438, 2431, 2024-08-25 16:59:31, 2024-08-25 16:59:38, 2024-08-25 17:40:09, 1724575209.V802I9c13e7M569645.mx2, carol@msa2.local
240, 2227, 1987, 1986, 2024-08-25 16:59:51, 2024-08-25 16:59:52, 2024-08-25 17:32:58, 1724574778.V802I9c11ceM957845.mx1, alice@msa1.local
250, 2830, 2580, 2579, 2024-08-25 17:00:01, 2024-08-25 17:00:02, 2024-08-25 17:43:01, 1724575381.V802I9c1499M584795.mx1, alice@msa1.local
370, 2830, 2460, 2445, 2024-08-25 17:02:01, 2024-08-25 17:02:16, 2024-08-25 17:43:01, 1724575381.V802I9c1498M584002.mx2, carol@msa2.local
380, 2831, 2451, 2450, 2024-08-25 17:02:11, 2024-08-25 17:02:12, 2024-08-25 17:43:02, 1724575382.V802I9c14aeM356992.mx2, carol@msa2.local`
	csv := strings.Join(data, "\n")
	if csv != expects {
		t.Errorf("csv expected %s, but got %s", expects, csv)
	}
}
