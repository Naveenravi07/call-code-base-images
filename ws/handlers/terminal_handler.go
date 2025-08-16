package handlers

import (
	"context"
	"fmt"
	"main/config"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type TerminalHandler struct {
	cfg *config.Config
}

func SetupTerminalHandler(cfg *config.Config) *TerminalHandler {
	return &TerminalHandler{cfg: cfg}
}

func (h *TerminalHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/terminal", h.terminalHandler)
}

func (h *TerminalHandler) terminalHandler(c *gin.Context) {
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	wsconn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Println("HTTP -> WS Upgrade error occured ")
		return
	}

	fmt.Println("Session name = ", h.cfg.SessionName)
	defer wsconn.Close()

	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Println("Failed to get cluster config")
		return
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println("K8 Client error")
		return
	}

	fmt.Println("Searching for pods with podname", h.cfg.SessionName)

	ct := context.Background()
	labelSelector := fmt.Sprintf("job-name=%s", h.cfg.SessionName)

	pods, err := clientset.CoreV1().Pods("default").List(ct, metav1.ListOptions{
		LabelSelector: labelSelector,
	})

	if err != nil {
		fmt.Printf("Error listing pods for job %s: %v\n", h.cfg.SessionName, err)
		return
	}
	if len(pods.Items) == 0 {
		fmt.Printf("No pods found under job %s\n", h.cfg.SessionName)
		return
	}

	var pod *v1.Pod
	for _, p := range pods.Items {
		if p.Status.Phase == v1.PodRunning {
			pod = &p
			break
		}
	}
	if pod == nil {
		fmt.Printf("No running pod found for job %s\n", h.cfg.SessionName)
		return
	}

	fmt.Printf("Found a pod %s running under job %s\n", pod.Name, h.cfg.SessionName)

	req := clientset.CoreV1().RESTClient().Post().Resource("pods").Name(pod.Name).Namespace("default").
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "user-service",
			Command:   []string{"sh"},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	exec, _ := remotecommand.NewSPDYExecutor(config, "POST", req.URL())

	err = exec.StreamWithContext(ct, remotecommand.StreamOptions{
		Stdin:  newWsReader(wsconn),
		Stdout: newWsWriter(wsconn),
		Stderr: newWsWriter(wsconn),
		Tty:    true,
	})

	if err != nil {
		fmt.Println("exec error:", err)
	}
}

type wsReader struct{ conn *websocket.Conn }

func newWsReader(c *websocket.Conn) *wsReader { return &wsReader{c} }
func (r *wsReader) Read(p []byte) (int, error) {
	_, msg, err := r.conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	n := copy(p, msg)
	return n, nil
}

type wsWriter struct{ conn *websocket.Conn }

func newWsWriter(c *websocket.Conn) *wsWriter { return &wsWriter{c} }
func (w *wsWriter) Write(p []byte) (int, error) {
	err := w.conn.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
