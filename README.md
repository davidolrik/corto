# Corto - Shorten all the Links 🔗

<img alt="Corto logo" width="200px;" src="assets/corto.png" align="right">

Corto is a modern and flexible link shortener written in Go.

## Features

- One short code multiple domains
- Click tracking
- Campaign tracking
- Device specific links
- Location statistics
- Add short link via API
- Nice WebIU

## Datamodel

```mermaid
erDiagram

Tenant {
  integer id
  integer owner_id "User"
  string name
}

User {
  integer id
  string username
  string password
}

Domain {
  integer id
  string fqdn
}

Shortcode {
  integer id
  string slug
  string target_url
  enum platform
}

Visit {
  integer id
  integer domain_id
  integer shortcode_id
  string ip_address
  string useragent
  string country
  string campaign
  timestamp create_dtm 
  timestamp update_dtm 
}

User ||--|{ Tenant : "Owns"

Tenant ||--o{ User : "Has many users"

User ||--o{ Domain : Owns
User }o--o{ Domain : "Has access to"

Domain }|--o{ Shortcode : Has

Domain ||--o{ Visit : "Came from"
Shortcode ||--o{ Visit : "Came from"

```
