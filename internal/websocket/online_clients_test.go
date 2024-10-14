package websocket

import "testing"

func TestNewOnlineClientsStoreCreation(t *testing.T) {
	oc := NewOnlineClientsStore()
	if oc == nil {
		t.Error("want OnlineClients{}, got nil")
	}

	if size := oc.Size(); size != 0 {
		t.Errorf("want %d, got %d", 0, size)
	}
}

func TestStoreClient(t *testing.T) {
	sessionId := "123"
	oc := NewOnlineClientsStore()
	cl := NewClient(sessionId, "text", []string{}, nil)
	oc.StoreClient(sessionId, cl)

	if size := oc.Size(); size != 1 {
		t.Errorf("want 'size' %d, got %d", 1, size)
	}

	v, ok := oc.Load(sessionId)
	if !ok {
		t.Errorf("want client 'sessionId'=%s to exist", sessionId)
	}

	if cl != v {
		t.Errorf("want %v, got %v", cl, v)
	}
}

func TestGetClient(t *testing.T) {
	sesh1, sesh2 := "1", "2"
	oc := NewOnlineClientsStore()
	cl := NewClient(sesh1, "text", []string{}, nil)
	oc.StoreClient(sesh1, cl)

	v, ok := oc.GetClient(sesh1)
	if !ok {
		t.Errorf("want client 'sessionId'=%s to exist", sesh1)
	}

	if cl != v {
		t.Errorf("want %v, got %v", cl, v)
	}

	v, ok = oc.GetClient(sesh2)
	if ok {
		t.Errorf("want client 'sessionId'=%s to not exist", sesh2)
	}

	if v != nil {
		t.Errorf("want nil, got %v", v)
	}
}

func TestDeleteClient(t *testing.T) {
	sesh1 := "1"
	oc := NewOnlineClientsStore()
	cl := NewClient(sesh1, "text", []string{}, nil)
	oc.StoreClient(sesh1, cl)

	oc.DeleteClient(sesh1)

	if size := oc.Size(); size != 0 {
		t.Errorf("want 'size' %d, got %d", 0, size)
	}
}

func TestHasCommonInterests(t *testing.T) {
	tests := []struct {
		name string
		want bool
		int1 []string
		int2 []string
	}{
		{
			name: "one common",
			want: true,
			int1: []string{"tech"},
			int2: []string{"tech", "film"},
		},
		{
			name: "zero common",
			want: false,
			int1: []string{"sports"},
			int2: []string{"tech", "film"},
		},
		{
			name: "both empty",
			want: false,
			int1: []string{},
			int2: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := hasCommonInterests(tt.int1, tt.int2)

			if res != tt.want {
				t.Errorf("want %v, got %v", tt.want, res)
			}
		})
	}
}

func TestCountCommonInterests(t *testing.T) {
	tests := []struct {
		name string
		want int
		int1 []string
		int2 []string
	}{
		{
			name: "one common",
			want: 1,
			int1: []string{"tech"},
			int2: []string{"tech", "film"},
		},
		{
			name: "zero common",
			want: 0,
			int1: []string{"sports"},
			int2: []string{"tech", "film"},
		},
		{
			name: "both empty",
			want: 0,
			int1: []string{},
			int2: []string{},
		},
		{
			name: "3 common",
			want: 3,
			int1: []string{"sports", "film", "college", "politics", "fitness"},
			int2: []string{"tech", "film", "fitness", "college"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := countCommonInterests(tt.int1, tt.int2)

			if res != tt.want {
				t.Errorf("want %d, got %d", tt.want, res)
			}
		})
	}
}
