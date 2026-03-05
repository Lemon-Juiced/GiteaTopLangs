# Gitea Top Langs
Gitea Top Langs aggregates language byte counts across all repositories for a given Gitea user and exports totals to JSON.  
It also prints a percentage breakdown to the console.  

**Prerequisites**
- Go 1.20+ installed  
- Access to a Gitea instance and a personal access token  

**Configuration**
Create a `config.json` in the program working directory.  
Example:  
```json
{
	"GITEA_URL": "http://gitea.example.local/",
	"GITEA_USER": "your-username",
	"GITEA_TOKEN": "your-access-token",
    "OUTPUT_DIR": "your-output-directory"
}
```

**Output**
- Console: prints each language total and percentage of the grand total.  
- If `OUTPUT_DIR` is provided in `config.json`, the program will create that directory (if needed) and write `languages_totals.json` into it.  
- If `OUTPUT_DIR` is omitted or empty, the program writes `languages_totals.json` into the current working directory where the program is run.  