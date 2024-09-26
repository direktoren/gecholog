package router

import (
	"encoding/json"
	"testing"

	"github.com/direktoren/gecholog/internal/protectedheader"
	"github.com/stretchr/testify/assert"
)

func Test_IngressNode_String(t *testing.T) {
	protectedheader.AddMaskedHeadersOnce([]string{"Masked-Header"})

	testCases := []struct {
		name     string
		nodeInfo IngressNode
		expected string
	}{
		{
			name:     "Empty NodeInfo",
			nodeInfo: IngressNode{},
			expected: "headers:map[]",
		},
		{
			name: "Filled NodeInfo",
			nodeInfo: IngressNode{
				Headers: protectedheader.ProtectedHeader{
					"Content-Type": []string{"application/json"},
				},
			},
			expected: "headers:map[Content-Type:[application/json]]",
		},
		{
			name: "Filled NodeInfo with Masked Header",
			nodeInfo: IngressNode{
				Headers: protectedheader.ProtectedHeader{
					"Masked-Header": []string{"application/json"},
				},
			},
			expected: "headers:map[Masked-Header:[*****MASKED*****]]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.nodeInfo.String()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func Test_OutboundNode_String(t *testing.T) {
	protectedheader.AddMaskedHeadersOnce([]string{"Masked-Header"})

	testCases := []struct {
		name     string
		nodeInfo OutboundNode
		expected string
	}{
		{
			name:     "Empty NodeInfo",
			nodeInfo: OutboundNode{},
			expected: "url: endpoint: headers:map[]",
		},
		{
			name: "Filled NodeInfo",
			nodeInfo: OutboundNode{
				Url:      "http://example.com",
				Endpoint: "/api/",
				Headers: protectedheader.ProtectedHeader{
					"Content-Type": []string{"application/json"},
				},
			},
			expected: "url:http://example.com endpoint:/api/ headers:map[Content-Type:[application/json]]",
		},
		{
			name: "Filled NodeInfo with Masked Header",
			nodeInfo: OutboundNode{
				Url:      "http://example.com",
				Endpoint: "/api/",
				Headers: protectedheader.ProtectedHeader{
					"Masked-Header": []string{"application/json"},
				},
			},
			expected: "url:http://example.com endpoint:/api/ headers:map[Masked-Header:[*****MASKED*****]]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.nodeInfo.String()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRouterString(t *testing.T) {
	protectedheader.AddMaskedHeadersOnce([]string{"Masked-Header"})

	// Test cases
	testCases := []struct {
		name        string
		router      Router
		expectedStr string
	}{
		{
			name:        "EmptyRouter",
			router:      Router{},
			expectedStr: "path: ingress:{headers:map[]} outbound:{url: endpoint: headers:map[]}",
		},
		{
			name: "RouterWithIngressAndOutbound",
			router: Router{
				Path: "/ingressPath/",
				Ingress: IngressNode{
					Headers: protectedheader.ProtectedHeader{
						"Ingress-Header": {"value1"},
					},
				},
				Outbound: OutboundNode{
					Url:      "http://outbound-example.com",
					Endpoint: "/outboundPath/",
					Headers: protectedheader.ProtectedHeader{
						"Outbound-Header": {"value2"},
					},
				},
			},
			expectedStr: "path:/ingressPath/ ingress:{headers:map[Ingress-Header:[value1]]} outbound:{url:http://outbound-example.com endpoint:/outboundPath/ headers:map[Outbound-Header:[value2]]}",
		},
		{
			name: "RouterWithIngressAndOutbound and Masked-Header",
			router: Router{
				Path: "/ingressPath/",
				Ingress: IngressNode{
					Headers: protectedheader.ProtectedHeader{
						"Masked-Header": {"value1"},
					},
				},
				Outbound: OutboundNode{
					Url:      "http://outbound-example.com",
					Endpoint: "/outboundPath/",
					Headers: protectedheader.ProtectedHeader{
						"Outbound-Header": {"value2"},
					},
				},
			},
			expectedStr: "path:/ingressPath/ ingress:{headers:map[Masked-Header:[*****MASKED*****]]} outbound:{url:http://outbound-example.com endpoint:/outboundPath/ headers:map[Outbound-Header:[value2]]}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedStr, tc.router.String())
		})
	}
}

func TestRouterMarshal(t *testing.T) {
	protectedheader.AddMaskedHeadersOnce([]string{"Masked-Header"})

	// Test cases
	testCases := []struct {
		name        string
		router      Router
		expectedStr string
	}{
		{
			name:        "EmptyRouter",
			router:      Router{},
			expectedStr: "{\"path\":\"\",\"ingress\":{\"headers\":{}},\"outbound\":{\"url\":\"\",\"endpoint\":\"\",\"headers\":{}}}"},
		{
			name: "RouterWithIngressAndOutbound",
			router: Router{
				Path: "/ingressPath/",
				Ingress: IngressNode{
					Headers: protectedheader.ProtectedHeader{
						"Ingress-Header": {"value1"},
					},
				},
				Outbound: OutboundNode{
					Url:      "http://outbound-example.com",
					Endpoint: "/outboundPath/",
					Headers: protectedheader.ProtectedHeader{
						"Outbound-Header": {"value2"},
					},
				},
			},
			expectedStr: "{\"path\":\"/ingressPath/\",\"ingress\":{\"headers\":{\"Ingress-Header\":[\"value1\"]}},\"outbound\":{\"url\":\"http://outbound-example.com\",\"endpoint\":\"/outboundPath/\",\"headers\":{\"Outbound-Header\":[\"value2\"]}}}",
		},
		{
			name: "RouterWithIngressAndOutbound and Masked-Header",
			router: Router{
				Path: "/ingressPath/",
				Ingress: IngressNode{
					Headers: protectedheader.ProtectedHeader{
						"Masked-Header": {"value1"},
					},
				},
				Outbound: OutboundNode{
					Url:      "http://outbound-example.com",
					Endpoint: "/outboundPath/",
					Headers: protectedheader.ProtectedHeader{
						"Outbound-Header": {"value2"},
					},
				},
			},
			expectedStr: "{\"path\":\"/ingressPath/\",\"ingress\":{\"headers\":{\"Masked-Header\":[\"*****MASKED*****\"]}},\"outbound\":{\"url\":\"http://outbound-example.com\",\"endpoint\":\"/outboundPath/\",\"headers\":{\"Outbound-Header\":[\"value2\"]}}}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bytes, err := json.Marshal(tc.router)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.expectedStr, string(bytes))
		})
	}
}
