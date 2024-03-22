package app

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"time"

	"github.com/yuin/goldmark"

	"bob-leaderboard/app/logger"
)

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type LinearIssuesResponse struct {
	Data struct {
		Team struct {
			Issues struct {
				Nodes []struct {
					Identifier      string `json:"identifier"`
					Title           string `json:"title"`
					Description     string `json:"description"`
					DescriptionHTML template.HTML
					State           struct {
						Name  string `json:"name"`
						Color string `json:"color"`
					} `json:"state"`
					Labels struct {
						Nodes []struct {
							Name  string `json:"name"`
							Color string `json:"color"`
						} `json:"nodes"`
					} `json:"labels"`
					CompletedAt time.Time   `json:"completedAt"`
					Parent      interface{} `json:"parent"`
				} `json:"nodes"`
			} `json:"issues"`
		} `json:"team"`
	} `json:"data"`
}

type Issue struct {
	Identifier      string `json:"identifier"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	DescriptionHTML template.HTML
	State           State       `json:"state"`
	Labels          []Label     `json:"labels"`
	CompletedAt     *time.Time  `json:"completedAt,omitempty"` // Pointer to handle nil (not completed) case
	Parent          interface{} `json:"parent"`
}

// Label representation for simplified structure.
type Label struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type State struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// StateGroup represents a group of issues by their state.
type StateGroup struct {
	Name  string  `json:"name"`
	Color string  `json:"color"`
	Items []Issue `json:"items"`
}

type OrganizedIssues []StateGroup

var IssuesData *OrganizedIssues = nil

func LoadAllIssues() error {
	// Construct the request payload
	// https://studio.apollographql.com/public/Linear-API/variant/current/explorer?explorerURLState=N4IgJg9gxgrgtgUwHYBcQC4QEcYIE4CeAFACQoICGcAkmOgAQDKKeAlkgOYCEANPSRDxh8AIQIMAChQ7sKKVhCQB5IaIJ8SAM1YAbcngbUAzkdwAxXfoCU9YAB0k9euSpFWdfi5pgb9x0-pWE1wjIkFhPDEGAVVI9XYoHRhhAEE8KAALVgA3BDpNCh0jBD5tPXxosutbBwCApAhhIxr-Oqd3ZHltfFq2p3kUHRLevqaoNgAHeUURtqMUOQQWvrqkKgRZvqgIHUFNgIBffacdCgAjBCLllacGpuub2-Xjtu3dvBenI9a275vtuATIbkMApFAvCYUPCdB4rDqoVjdD4-Op-X6zNHfb4gA5AA
	payload := GraphQLRequest{
		Query: `query($teamId: String!, $orderBy: PaginationOrderBy, $filter: IssueFilter) {
			team(id: $teamId) {
				issues(orderBy: $orderBy,includeArchived:false, filter: $filter) {
					nodes {
						identifier
						title
						description
						state {
						  name
						  color
						}
						labels {
							nodes {
								name
								color
							}
						}
						completedAt
						parent {
							identifier
						}
					}
				}
			}
		}`,
		Variables: map[string]interface{}{
			"teamId":  "BAS",
			"orderBy": "updatedAt",
			"filter": map[string]interface{}{
				"parent": map[string]interface{}{
					"null": true,
				},
			},
		},
	}

	// Marshal the payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Error("Error marshalling payload: %v", err)
		return err
	}

	// Create a new HTTP request
	req, err := http.NewRequest("POST", "https://api.linear.app/graphql", bytes.NewBuffer(payloadBytes))
	if err != nil {
		logger.Error("Error creating request: %v", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", Config.GetString("LinearApiKey"))

	// Make the request using the default client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Error making request: %v", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// read the response body as string
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Error("Error reading response body: %v", err)
			return err
		}

		logger.Error("Error response: %v", string(body))

		return err
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Error reading response body: %v", err)
		return err
	}

	var response LinearIssuesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		logger.Error("Error unmarshalling response: %v", err)
		return err
	}

	OrganizeIssues(response)

	return nil
}

var StateOrder = []string{"Backlog", "Todo", "In Progress", "Done", "Canceled", "Duplicate"}
var StateColors = []string{"#95a2b3", "#e2e2e2", "#f2c94c", "#5e6ad2", "#95a2b3", "#95a2b3"}

func OrganizeIssues(response LinearIssuesResponse) {

	organized := make(OrganizedIssues, len(StateOrder))
	stateIndexMap := make(map[string]int)
	for i, stateName := range StateOrder {
		organized[i] = StateGroup{
			Name:  stateName,
			Color: StateColors[i],
			Items: []Issue{},
		}
		stateIndexMap[stateName] = i
	}

	for _, node := range response.Data.Team.Issues.Nodes {
		// Get the index of the current state from the map
		index, exists := stateIndexMap[node.State.Name]
		if !exists {
			// Handle unknown state; could log a warning or dynamically add new states if needed
			continue
		}

		// Update state color if necessary
		if organized[index].Color == "" {
			organized[index].Color = node.State.Color
		}

		// Prepare labels for the simplified issue
		labels := make([]Label, len(node.Labels.Nodes))
		for j, label := range node.Labels.Nodes {
			labels[j] = Label{
				Name:  label.Name,
				Color: label.Color,
			}
		}

		// Construct the simplified issue
		issue := Issue{
			Identifier:  node.Identifier,
			Title:       node.Title,
			Description: node.Description,
			Labels:      labels,
			Parent:      node.Parent,
		}
		descriptionHTML, err := ConvertMarkdownToHTML([]byte(node.Description))
		if err != nil {
			logger.Error("Error converting markdown to HTML: %v", err)
		} else {
			issue.DescriptionHTML = template.HTML(descriptionHTML)
		}

		if !node.CompletedAt.IsZero() {
			issue.CompletedAt = &node.CompletedAt
		}

		// Append the issue to the appropriate state group
		organized[index].Items = append(organized[index].Items, issue)
	}

	IssuesData = &organized
}

func ConvertMarkdownToHTML(markdown []byte) (string, error) {
	var buf bytes.Buffer
	if err := goldmark.Convert(markdown, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
