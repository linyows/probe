# Data Flow

Data flow is the mechanism by which information moves through Probe workflows. Understanding data flow patterns enables you to build sophisticated workflows that pass information between steps, jobs, and even external systems. This guide explores the complete data flow system in Probe.

## Data Flow Overview

Probe uses a structured approach to data flow:

1. **Input Sources**: Environment variables, configuration files, user inputs
2. **Processing**: Actions generate responses and outputs
3. **Storage**: Outputs are stored for later use
4. **Propagation**: Data flows between steps and jobs
5. **Consumption**: Other steps use the data for dynamic configuration

## Data Sources

### Environment Variables

Environment variables provide external configuration and runtime context.

```yaml
# Access environment variables
steps:
  - name: Environment-based Configuration
    action: http
    with:
      url: "{{env.API_BASE_URL}}/{{env.API_VERSION}}/users"
      headers:
        Authorization: "Bearer {{env.API_TOKEN}}"
        X-Environment: "{{env.DEPLOYMENT_ENV}}"
    test: res.status == 200
```

### Configuration Merging

Data can come from merged configuration files:

**base-config.yml:**
```yaml
defaults:
  api:
    timeout: 30s
    retry_count: 3
env:
  API_BASE_URL: https://api.example.com
```

**production.yml:**
```yaml
env:
  API_BASE_URL: https://api.production.example.com
  API_TOKEN: ${PROD_API_TOKEN}
defaults:
  api:
    timeout: 10s  # Override for production
```

Usage:
```bash
probe workflow.yml,base-config.yml,production.yml
```

## Step Outputs

Steps generate outputs that can be consumed by subsequent steps and jobs.

### Basic Output Definition

```yaml
steps:
  - name: User Authentication
    id: auth
    action: http
    with:
      url: "{{env.API_URL}}/auth/login"
      method: POST
      body: |
        {
          "username": "{{env.USERNAME}}",
          "password": "{{env.PASSWORD}}"
        }
    test: res.status == 200
    outputs:
      access_token: res.json.access_token
      refresh_token: res.json.refresh_token
      user_id: res.json.user.id
      expires_at: res.json.expires_at
      user_roles: res.json.user.roles
```

### Output Data Types

Outputs can contain various data types:

```yaml
- name: Comprehensive Data Collection
  id: data-collection
  action: http
  with:
    url: "{{env.API_URL}}/comprehensive-data"
  test: res.status == 200
  outputs:
    # Simple values
    user_count: res.json.stats.user_count
    server_version: res.json.version
    is_healthy: res.json.health.status == "healthy"
    
    # Complex objects
    user_profile: res.json.user
    configuration: res.json.config
    metrics: res.json.metrics
    
    # Arrays
    active_users: res.json.users.filter(u -> u.active == true)
    error_codes: res.json.errors.map(e -> e.code)
    
    # Computed values
    success_rate: (res.json.successful_requests / res.json.total_requests) * 100
    avg_response_time: res.json.response_times.sum() / res.json.response_times.length
    
    # Response metadata
    response_time: res.time
    response_size: res.body_size
    content_type: res.headers["content-type"]
```

### Output Scoping

Outputs are scoped to their containing step and can be referenced by ID:

```yaml
steps:
  - name: Database Setup
    id: db-setup
    action: http
    with:
      url: "{{env.DB_API}}/initialize"
    outputs:
      db_session_id: res.json.session_id
      db_host: res.json.host
      db_port: res.json.port

  - name: Application Test
    id: app-test
    action: http
    with:
      url: "{{env.APP_URL}}/test"
      headers:
        X-DB-Session: "{{outputs.db-setup.db_session_id}}"
        X-DB-Host: "{{outputs.db-setup.db_host}}"
    outputs:
      test_result: res.json.result
      test_duration: res.time

  - name: Performance Analysis
    echo: |
      Performance Analysis:
      Database: {{outputs.db-setup.db_host}}:{{outputs.db-setup.db_port}}
      Test Result: {{outputs.app-test.test_result}}
      Test Duration: {{outputs.app-test.test_duration}}ms
```

## Cross-Job Data Flow

Data can flow between jobs through job-level outputs and dependencies.

### Job Dependencies and Data Sharing

```yaml
jobs:
  initialization:
    name: System Initialization
    steps:
      - name: Create Test Environment
        id: env-setup
        action: http
        with:
          url: "{{env.SETUP_API}}/create-environment"
          method: POST
          body: |
            {
              "environment_name": "test_{{random_str(8)}}",
              "configuration": "standard"
            }
        test: res.status == 201
        outputs:
          environment_id: res.json.environment.id
          environment_name: res.json.environment.name
          database_url: res.json.environment.database_url
          api_endpoint: res.json.environment.api_endpoint

  api-tests:
    name: API Testing Suite
    needs: [initialization]  # Wait for initialization to complete
    steps:
      - name: Test User API
        action: http
        with:
          url: "{{outputs.initialization.api_endpoint}}/users"
          headers:
            X-Environment: "{{outputs.initialization.environment_id}}"
        test: res.status == 200
        outputs:
          user_count: res.json.total_users
          api_response_time: res.time

      - name: Test Database Connectivity
        action: http
        with:
          url: "{{outputs.initialization.database_url}}/ping"
        test: res.status == 200
        outputs:
          db_response_time: res.time

  reporting:
    name: Test Reporting
    needs: [initialization, api-tests]  # Wait for both jobs
    steps:
      - name: Generate Test Report
        echo: |
          Test Execution Report
          =====================
          
          Environment: {{outputs.initialization.environment_name}}
          Environment ID: {{outputs.initialization.environment_id}}
          
          API Tests:
          - User Count: {{outputs.api-tests.user_count}}
          - API Response Time: {{outputs.api-tests.api_response_time}}ms
          
          Database Tests:
          - DB Response Time: {{outputs.api-tests.db_response_time}}ms
          
          Overall Status: All tests completed successfully

  cleanup:
    name: Environment Cleanup
    needs: [reporting]  # Run after reporting completes
    steps:
      - name: Destroy Test Environment
        action: http
        with:
          url: "{{env.SETUP_API}}/environments/{{outputs.initialization.environment_id}}"
          method: DELETE
        test: res.status == 204
```

### Cross-Job Output References

Access outputs from other jobs using the `outputs.job-name.output-name` syntax:

```yaml
jobs:
  data-collection:
    steps:
      - name: Collect User Data
        outputs:
          total_users: res.json.count
          active_users: res.json.active_count

  analysis:
    needs: [data-collection]
    steps:
      - name: Analyze User Metrics
        echo: |
          User Analysis:
          Total Users: {{outputs.data-collection.total_users}}
          Active Users: {{outputs.data-collection.active_users}}
          Activity Rate: {{(outputs.data-collection.active_users / outputs.data-collection.total_users) * 100}}%
```

## Advanced Data Flow Patterns

### Data Transformation Chains

Transform data through multiple steps:

```yaml
jobs:
  data-processing-pipeline:
    name: Data Processing Pipeline
    steps:
      - name: Fetch Raw Data
        id: raw-data
        action: http
        with:
          url: "{{env.DATA_API}}/raw-data"
        outputs:
          raw_records: res.json.records
          total_count: res.json.total
          fetch_time: res.time

      - name: Filter Data
        id: filtered-data
        echo: "Filtering data..."
        outputs:
          # Filter active records
          active_records: "{{outputs.raw-data.raw_records.filter(r -> r.status == 'active')}}"
          active_count: "{{outputs.raw-data.raw_records.filter(r -> r.status == 'active').length}}"
          
      - name: Aggregate Data
        id: aggregated-data
        echo: "Aggregating data..."
        outputs:
          # Group by category and calculate metrics
          categories: "{{outputs.filtered-data.active_records.groupBy(r -> r.category)}}"
          avg_score: "{{outputs.filtered-data.active_records.map(r -> r.score).sum() / outputs.filtered-data.active_count}}"
          
      - name: Generate Summary
        echo: |
          Data Processing Summary:
          
          Raw Records: {{outputs.raw-data.total_count}}
          Active Records: {{outputs.filtered-data.active_count}}
          Processing Rate: {{(outputs.filtered-data.active_count / outputs.raw-data.total_count) * 100}}%
          Average Score: {{outputs.aggregated-data.avg_score}}
          Fetch Time: {{outputs.raw-data.fetch_time}}ms
```

### Conditional Data Flow

Control data flow based on conditions:

```yaml
jobs:
  adaptive-processing:
    steps:
      - name: Assess Data Quality
        id: quality-check
        action: http
        with:
          url: "{{env.API_URL}}/data-quality"
        outputs:
          quality_score: res.json.quality_score
          has_errors: res.json.error_count > 0
          record_count: res.json.record_count
          
      - name: Standard Processing
        if: outputs.quality-check.quality_score >= 0.8 && !outputs.quality-check.has_errors
        id: standard-processing
        action: http
        with:
          url: "{{env.PROCESSING_API}}/standard"
          method: POST
          body: |
            {
              "record_count": {{outputs.quality-check.record_count}},
              "quality_mode": "standard"
            }
        outputs:
          processing_result: res.json.result
          processing_time: res.time
          
      - name: Enhanced Processing
        if: outputs.quality-check.quality_score < 0.8 || outputs.quality-check.has_errors
        id: enhanced-processing
        action: http
        with:
          url: "{{env.PROCESSING_API}}/enhanced"
          method: POST
          body: |
            {
              "record_count": {{outputs.quality-check.record_count}},
              "quality_mode": "enhanced",
              "error_correction": true
            }
        outputs:
          processing_result: res.json.result
          processing_time: res.time
          corrections_applied: res.json.corrections
          
      - name: Processing Summary
        echo: |
          Data Processing Complete:
          
          Quality Score: {{outputs.quality-check.quality_score}}
          Processing Mode: {{outputs.quality-check.quality_score >= 0.8 ? "Standard" : "Enhanced"}}
          
          {{outputs.standard-processing ? "Standard Processing Time: " + outputs.standard-processing.processing_time + "ms" : ""}}
          {{outputs.enhanced-processing ? "Enhanced Processing Time: " + outputs.enhanced-processing.processing_time + "ms" : ""}}
          {{outputs.enhanced-processing ? "Corrections Applied: " + outputs.enhanced-processing.corrections_applied : ""}}
```

### Data Accumulation Patterns

Collect data from multiple sources:

```yaml
jobs:
  multi-source-data-collection:
    steps:
      - name: Source A Data
        id: source-a
        action: http
        with:
          url: "{{env.SOURCE_A_URL}}/data"
        outputs:
          source_a_count: res.json.count
          source_a_data: res.json.data
          source_a_time: res.time

      - name: Source B Data
        id: source-b
        action: http
        with:
          url: "{{env.SOURCE_B_URL}}/data"
        outputs:
          source_b_count: res.json.count
          source_b_data: res.json.data
          source_b_time: res.time

      - name: Source C Data
        id: source-c
        action: http
        with:
          url: "{{env.SOURCE_C_URL}}/data"
        outputs:
          source_c_count: res.json.count
          source_c_data: res.json.data
          source_c_time: res.time

      - name: Aggregate All Sources
        echo: |
          Multi-Source Data Summary:
          
          Source A: {{outputs.source-a.source_a_count}} records ({{outputs.source-a.source_a_time}}ms)
          Source B: {{outputs.source-b.source_b_count}} records ({{outputs.source-b.source_b_time}}ms)
          Source C: {{outputs.source-c.source_c_count}} records ({{outputs.source-c.source_c_time}}ms)
          
          Total Records: {{outputs.source-a.source_a_count + outputs.source-b.source_b_count + outputs.source-c.source_c_count}}
          Average Response Time: {{(outputs.source-a.source_a_time + outputs.source-b.source_b_time + outputs.source-c.source_c_time) / 3}}ms
          
          Fastest Source: {{
            outputs.source-a.source_a_time <= outputs.source-b.source_b_time && outputs.source-a.source_a_time <= outputs.source-c.source_c_time ? "Source A" :
            outputs.source-b.source_b_time <= outputs.source-c.source_c_time ? "Source B" : "Source C"
          }}
```

## Data Validation and Quality

### Output Validation

Ensure data quality in outputs:

```yaml
- name: Data Collection with Validation
  id: validated-data
  action: http
  with:
    url: "{{env.API_URL}}/user-data"
  test: |
    res.status == 200 &&
    res.json.users != null &&
    res.json.users.length > 0 &&
    res.json.users.all(u -> u.id != null && u.email != null)
  outputs:
    # Validated outputs
    user_count: res.json.users.length
    valid_users: res.json.users.filter(u -> u.id != null && u.email != null)
    admin_users: res.json.users.filter(u -> u.role == "admin")
    
    # Data quality metrics
    data_completeness: res.json.users.filter(u -> u.id != null && u.email != null).length / res.json.users.length
    has_admin_users: res.json.users.any(u -> u.role == "admin")
    
    # Response metadata
    data_freshness: res.headers["last-modified"]
    cache_status: res.headers["x-cache-status"]
```

### Data Sanitization

Clean and sanitize data before use:

```yaml
- name: Sanitize User Input
  id: sanitized-input
  action: http
  with:
    url: "{{env.API_URL}}/user-input"
  outputs:
    # Raw data
    raw_input: res.json.input
    
    # Sanitized data
    clean_email: res.json.input.email.lower().trim()
    clean_name: res.json.input.name.trim()
    safe_description: res.json.input.description.substring(0, 500)  # Limit length
    
    # Validation flags
    email_valid: res.json.input.email.matches("[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}")
    name_valid: res.json.input.name.length >= 2 && res.json.input.name.length <= 50
```

## Performance Considerations

### Efficient Data Access

Optimize data access patterns:

```yaml
# Good: Direct property access
outputs:
  user_id: res.json.user.id
  user_name: res.json.user.name

# Good: Single computation with reuse
outputs:
  active_users: res.json.users.filter(u -> u.active == true)
  active_user_count: res.json.users.filter(u -> u.active == true).length

# Avoid: Repeated expensive computations
# outputs:
#   user_count: res.json.users.filter(u -> expensive_validation(u)).length
#   user_list: res.json.users.filter(u -> expensive_validation(u))
```

### Memory Management

Be mindful of large data sets:

```yaml
# Good: Extract essential data only
outputs:
  user_ids: res.json.users.map(u -> u.id)
  user_count: res.json.users.length
  first_user: res.json.users[0]

# Avoid: Storing large objects unnecessarily
# outputs:
#   all_user_data: res.json.users  # Could be very large
#   complete_response: res.json     # Entire response body
```

### Selective Data Extraction

Extract only needed data:

```yaml
- name: Efficient Data Extraction
  action: http
  with:
    url: "{{env.API_URL}}/large-dataset"
  outputs:
    # Extract summary information only
    record_count: res.json.metadata.total_records
    last_updated: res.json.metadata.last_updated
    status: res.json.metadata.status
    
    # Extract specific records by criteria
    critical_items: res.json.data.filter(item -> item.priority == "critical")
    error_items: res.json.data.filter(item -> item.status == "error")
    
    # Compute aggregates
    avg_score: res.json.data.map(item -> item.score).sum() / res.json.data.length
    max_score: res.json.data.map(item -> item.score).max()
    
    # Don't store the entire dataset
    # full_dataset: res.json.data  # Avoid this for large datasets
```

## Best Practices

### 1. Clear Output Naming

Use descriptive names for outputs:

```yaml
# Good: Descriptive names
outputs:
  user_authentication_token: res.json.access_token
  session_expiry_timestamp: res.json.expires_at
  user_permission_level: res.json.user.role

# Avoid: Generic names
outputs:
  token: res.json.access_token
  time: res.json.expires_at
  level: res.json.user.role
```

### 2. Type-Consistent Outputs

Maintain consistent data types:

```yaml
# Good: Consistent types
outputs:
  user_count: res.json.users.length          # Always number
  is_admin: res.json.user.role == "admin"    # Always boolean
  user_email: res.json.user.email || ""      # Always string (with default)

# Avoid: Inconsistent types
outputs:
  user_count: res.json.users.length || "unknown"  # Number or string
```

### 3. Error-Safe Data Access

Handle potential null/undefined values:

```yaml
# Good: Safe data access
outputs:
  user_id: res.json.user && res.json.user.id ? res.json.user.id : null
  email_verified: res.json.user && res.json.user.email_verified == true
  profile_complete: res.json.user && res.json.user.profile && res.json.user.profile.complete == true

# Good: Using safe navigation
test: res.json.user?.id != null && res.json.user?.email != null
```

### 4. Document Data Dependencies

Document what data flows where:

```yaml
jobs:
  user-setup:
    name: User Account Setup
    steps:
      - name: Create User Account
        # Produces: user_id, username, email
        outputs:
          user_id: res.json.user.id
          username: res.json.user.username
          email: res.json.user.email

  user-verification:
    name: User Account Verification
    needs: [user-setup]
    steps:
      - name: Send Verification Email
        # Consumes: user_id, email from user-setup
        action: smtp
        with:
          to: ["{{outputs.user-setup.email}}"]
          subject: "Verify your account"
          body: "Click here to verify user {{outputs.user-setup.user_id}}"
```

## What's Next?

Now that you understand data flow, explore:

1. **[Testing and Assertions](../testing-and-assertions/)** - Learn validation techniques
2. **[Error Handling](../error-handling/)** - Handle data flow failures gracefully  
3. **[How-tos](../../how-tos/)** - See practical data flow patterns

Data flow is the circulatory system of your Probe workflows. Master these patterns to build sophisticated, data-driven automation processes.