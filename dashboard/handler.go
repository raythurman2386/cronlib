package dashboard

import (
	"encoding/json"
	"net/http"
	"text/template"

	"github.com/raythurman2386/cronlib"
)

// Handler serves the dashboard.
type Handler struct {
	client *cronlib.Cron
}

// NewHandler creates a new handler.
func NewHandler(c *cronlib.Cron) http.Handler {
	mux := http.NewServeMux()
	h := &Handler{client: c}
	mux.HandleFunc("/", h.index)
	mux.HandleFunc("/api/jobs", h.listJobs)
	return mux
}

func (h *Handler) listJobs(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for development
	w.Header().Set("Access-Control-Allow-Origin", "*")

	jobs := h.client.GetJobs()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(jobs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) index(w http.ResponseWriter, r *http.Request) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
	<title>CronLib Dashboard</title>
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; padding: 20px; background: #f4f4f9; color: #333; }
		h1 { margin-bottom: 20px; }
		table { width: 100%; border-collapse: collapse; background: white; box-shadow: 0 1px 3px rgba(0,0,0,0.1); border-radius: 4px; overflow: hidden; }
		th, td { padding: 12px 15px; text-align: left; border-bottom: 1px solid #eee; }
		th { background-color: #fafafa; font-weight: 600; text-transform: uppercase; font-size: 0.85em; letter-spacing: 0.5px; color: #555; }
		tr:last-child td { border-bottom: none; }
		tr:hover { background-color: #f9f9f9; }
		.running { color: green; font-weight: bold; }
		.idle { color: #888; }
	</style>
</head>
<body>
	<h1>CronLib Dashboard</h1>
	<table id="jobs">
		<thead>
			<tr>
				<th>ID</th>
				<th>Expression</th>
				<th>Next Run</th>
				<th>Last Run</th>
				<th>Status</th>
			</tr>
		</thead>
		<tbody></tbody>
	</table>
	<script>
		function load() {
			fetch('/api/jobs')
				.then(response => response.json())
				.then(data => {
					const body = document.querySelector('#jobs tbody');
					body.innerHTML = '';
					data.forEach(job => {
						const row = document.createElement('tr');
						const statusClass = job.running ? 'running' : 'idle';
						const lastRun = job.last_run === "0001-01-01T00:00:00Z" ? "Never" : new Date(job.last_run).toLocaleString();
						
						row.innerHTML = '<td>' + job.id + '</td>' +
										'<td>' + job.expression + '</td>' +
										'<td>' + new Date(job.next_run).toLocaleString() + '</td>' +
										'<td>' + lastRun + '</td>' +
										'<td class="' + statusClass + '">' + (job.running ? "Running" : "Idle") + '</td>';
						body.appendChild(row);
					});
				});
		}
		load();
		setInterval(load, 2000);
	</script>
</body>
</html>
`
	t, err := template.New("index").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = t.Execute(w, nil)
}
