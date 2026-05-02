package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/secrets"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage environment variables / secrets",
}

var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all secrets for a company",
	RunE:  runEnvList,
}

var envSetCmd = &cobra.Command{
	Use:   "set KEY VALUE",
	Short: "Set a secret",
	Args:  cobra.ExactArgs(2),
	RunE:  runEnvSet,
}

var envGetCmd = &cobra.Command{
	Use:   "get KEY",
	Short: "Get a secret value",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvGet,
}

var (
	flagEnvCompany string
	flagEnvUseDB   bool
)

func init() {
	envListCmd.Flags().StringVar(&flagEnvCompany, "company", "", "Company ID (required)")
	_ = envListCmd.MarkFlagRequired("company")

	envSetCmd.Flags().StringVar(&flagEnvCompany, "company", "", "Company ID (required)")
	envSetCmd.Flags().BoolVar(&flagEnvUseDB, "db", false, "Use direct database access instead of HTTP")
	_ = envSetCmd.MarkFlagRequired("company")

	envGetCmd.Flags().StringVar(&flagEnvCompany, "company", "", "Company ID (required)")
	envGetCmd.Flags().BoolVar(&flagEnvUseDB, "db", false, "Use direct database access instead of HTTP")
	_ = envGetCmd.MarkFlagRequired("company")

	envCmd.AddCommand(envListCmd, envSetCmd, envGetCmd)
	rootCmd.AddCommand(envCmd)
}

func runEnvList(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Try HTTP first
	client, err := NewHTTPClient()
	if err == nil {
		defer client.Close()
		return listViaHTTP(ctx, client, flagEnvCompany)
	}

	// Fallback to database
	return listViaDB(ctx, flagEnvCompany)
}

func listViaHTTP(ctx context.Context, client *HTTPClient, companyID string) error {
	query := url.QueryEscape(companyID)
	req, err := http.NewRequestWithContext(ctx, "GET",
		client.BaseURL()+"/api/secrets?companyId="+query, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("requesting secrets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Items []*domain.SecretSummary `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	// Pretty-print names with creation dates
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tCREATED_AT")
	for _, s := range result.Items {
		fmt.Fprintf(w, "%s\t%s\n", s.Name, s.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	w.Flush()
	return nil
}

func listViaDB(ctx context.Context, companyID string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	svc := secrets.New(s)
	items, err := svc.ListByCompany(ctx, companyID)
	if err != nil {
		return fmt.Errorf("listing secrets: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tCREATED_AT")
	for _, s := range items {
		fmt.Fprintf(w, "%s\t%s\n", s.Name, s.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	w.Flush()
	return nil
}

func runEnvSet(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	name := args[0]
	value := args[1]

	if flagEnvUseDB {
		return setViaDB(ctx, flagEnvCompany, name, value)
	}

	client, err := NewHTTPClient()
	if err != nil {
		// If HTTP client setup fails, fall back to DB
		return setViaDB(ctx, flagEnvCompany, name, value)
	}
	defer client.Close()

	return setViaHTTP(ctx, client, flagEnvCompany, name, value)
}

func setViaHTTP(ctx context.Context, client *HTTPClient, companyID, name, value string) error {
	body := struct {
		CompanyID string `json:"companyId"`
		Name      string `json:"name"`
		Value     string `json:"value"`
	}{
		CompanyID: companyID,
		Name:      name,
		Value:     value,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		client.BaseURL()+"/api/secrets",
		strings.NewReader(string(bodyBytes)))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("posting secret: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var secret *domain.Secret
	if err := json.NewDecoder(resp.Body).Decode(&secret); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	fmt.Printf("Secret '%s' created with ID %s\n", secret.Name, secret.ID)
	return nil
}

func setViaDB(ctx context.Context, companyID, name, value string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	svc := secrets.New(s)
	secret, err := svc.Create(ctx, companyID, name, value)
	if err != nil {
		return fmt.Errorf("creating secret: %w", err)
	}

	fmt.Printf("Secret '%s' created with ID %s\n", secret.Name, secret.ID)
	return nil
}

func runEnvGet(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	name := args[0]

	if flagEnvUseDB {
		return getViaDB(ctx, flagEnvCompany, name)
	}

	client, err := NewHTTPClient()
	if err != nil {
		// If HTTP client setup fails, fall back to DB
		return getViaDB(ctx, flagEnvCompany, name)
	}
	defer client.Close()

	return getViaHTTP(ctx, client, flagEnvCompany, name)
}

func getViaHTTP(ctx context.Context, client *HTTPClient, companyID, name string) error {
	// List secrets to find by name
	query := url.QueryEscape(companyID)
	req, err := http.NewRequestWithContext(ctx, "GET",
		client.BaseURL()+"/api/secrets?companyId="+query, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("requesting secrets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Items []*domain.SecretSummary `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	// Find secret by name
	var secretID string
	for _, s := range result.Items {
		if s.Name == name {
			secretID = s.ID
			break
		}
	}

	if secretID == "" {
		return fmt.Errorf("secret %q not found", name)
	}

	// Get the full secret with value
	req2, err := http.NewRequestWithContext(ctx, "GET",
		client.BaseURL()+"/api/secrets/"+url.QueryEscape(secretID), nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp2, err := client.Do(req2)
	if err != nil {
		return fmt.Errorf("requesting secret: %w", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp2.Body)
		return fmt.Errorf("HTTP %d: %s", resp2.StatusCode, string(body))
	}

	var secret *domain.Secret
	if err := json.NewDecoder(resp2.Body).Decode(&secret); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	fmt.Println(secret.Value)
	return nil
}

func getViaDB(ctx context.Context, companyID, name string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	svc := secrets.New(s)
	items, err := svc.ListByCompany(ctx, companyID)
	if err != nil {
		return fmt.Errorf("listing secrets: %w", err)
	}

	// Find by name
	var secretID string
	for _, s := range items {
		if s.Name == name {
			secretID = s.ID
			break
		}
	}

	if secretID == "" {
		return fmt.Errorf("secret %q not found", name)
	}

	// Get full secret
	secret, err := svc.GetByID(ctx, secretID)
	if err != nil {
		return fmt.Errorf("getting secret: %w", err)
	}

	fmt.Println(secret.Value)
	return nil
}
