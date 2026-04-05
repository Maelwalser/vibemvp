# Ansible Skill Guide

## Overview

Ansible automates configuration management, app deployment, and orchestration via idempotent playbooks. Agentless — uses SSH (Linux) or WinRM (Windows).

## Playbook Structure

```yaml
# playbook.yml
---
- name: Deploy API service
  hosts: api_servers
  become: true   # sudo
  vars:
    app_user: appuser
    app_dir: /opt/api
    app_version: "{{ lookup('env', 'APP_VERSION') | default('latest') }}"

  pre_tasks:
    - name: Update apt cache
      apt:
        update_cache: true
        cache_valid_time: 3600
      when: ansible_os_family == "Debian"

  roles:
    - common
    - api

  post_tasks:
    - name: Verify service is running
      uri:
        url: http://localhost:8080/healthz
        status_code: 200
      retries: 5
      delay: 3
```

## Role Structure

```
roles/api/
├── tasks/
│   └── main.yml       # Task list
├── handlers/
│   └── main.yml       # Handlers (triggered by notify)
├── templates/
│   └── app.conf.j2    # Jinja2 config templates
├── files/
│   └── logrotate.conf # Static files to copy
├── defaults/
│   └── main.yml       # Default variable values
└── vars/
    └── main.yml       # Role-specific overrides (higher priority)
```

```yaml
# roles/api/tasks/main.yml
---
- name: Create app user
  user:
    name: "{{ app_user }}"
    system: true
    shell: /bin/false
    create_home: false

- name: Create app directory
  file:
    path: "{{ app_dir }}"
    state: directory
    owner: "{{ app_user }}"
    mode: "0750"

- name: Copy application binary
  copy:
    src: "files/api-{{ app_version }}"
    dest: "{{ app_dir }}/api"
    owner: "{{ app_user }}"
    mode: "0750"
  notify: Restart api   # triggers handler

- name: Deploy config from template
  template:
    src: app.conf.j2
    dest: "{{ app_dir }}/config.yaml"
    owner: "{{ app_user }}"
    mode: "0640"
  notify: Restart api

- name: Install systemd unit
  template:
    src: api.service.j2
    dest: /etc/systemd/system/api.service
    mode: "0644"
  notify:
    - Reload systemd
    - Restart api

- name: Ensure api service is enabled and started
  systemd:
    name: api
    enabled: true
    state: started
    daemon_reload: true
```

```yaml
# roles/api/handlers/main.yml
---
- name: Reload systemd
  systemd:
    daemon_reload: true

- name: Restart api
  systemd:
    name: api
    state: restarted
```

```jinja2
{# roles/api/templates/app.conf.j2 #}
app_env: {{ app_env | default('production') }}
port: {{ app_port | default(8080) }}
database_url: {{ database_url }}
log_level: {{ log_level | default('info') }}
```

## Inventory

```ini
# inventory/hosts.ini
[api_servers]
api-01.example.com ansible_user=ubuntu
api-02.example.com ansible_user=ubuntu

[db_servers]
db-01.example.com ansible_user=ubuntu

[all:vars]
ansible_ssh_private_key_file=~/.ssh/deploy_key
```

```yaml
# inventory/hosts.yml (YAML format)
all:
  children:
    api_servers:
      hosts:
        api-01.example.com:
          ansible_user: ubuntu
        api-02.example.com:
          ansible_user: ubuntu
      vars:
        app_port: 8080
    db_servers:
      hosts:
        db-01.example.com:
          ansible_user: ubuntu
```

## Ansible Vault (secret encryption)

```bash
# Encrypt a string inline
ansible-vault encrypt_string 'mysecretpassword' --name 'db_password'
# Paste output into vars file

# Encrypt a file
ansible-vault encrypt group_vars/prod/secrets.yml

# Edit encrypted file
ansible-vault edit group_vars/prod/secrets.yml

# Run playbook with vault password
ansible-playbook playbook.yml --ask-vault-pass
# Or use password file:
ansible-playbook playbook.yml --vault-password-file ~/.vault_pass
```

```yaml
# group_vars/prod/secrets.yml (encrypted with vault)
db_password: !vault |
  $ANSIBLE_VAULT;1.1;AES256
  61663930...
```

## Idempotent Task Patterns

```yaml
# Use creates:/removes: for shell/command tasks
- name: Extract archive
  command: tar xzf /tmp/app.tar.gz -C /opt/app
  args:
    creates: /opt/app/bin/api   # skip if file exists

- name: Remove old config
  file:
    path: /etc/app/old.conf
    state: absent
  args:
    removes: /etc/app/old.conf  # skip if already absent

# Use when: for conditional tasks
- name: Initialize database
  command: /opt/app/api db:init
  when: db_initialized.stat.exists == false

- name: Check if DB initialized
  stat:
    path: /opt/app/.db_initialized
  register: db_initialized
```

## Running Playbooks

```bash
# Check syntax
ansible-playbook playbook.yml --syntax-check

# Dry run
ansible-playbook playbook.yml --check --diff

# Limit to specific hosts
ansible-playbook playbook.yml --limit api-01.example.com

# Run specific tags
ansible-playbook playbook.yml --tags deploy

# Pass extra variables
ansible-playbook playbook.yml -e "app_version=1.2.3 app_env=production"

# Ad-hoc command
ansible api_servers -m ping -i inventory/hosts.ini
ansible api_servers -m systemd -a "name=api state=restarted" --become
```

## Key Rules

- All tasks must be idempotent — running twice must produce the same result.
- Use `notify` + handlers for service restarts — handlers run once at the end even if notified multiple times.
- Use `become: true` at play or task level for privilege escalation, not running Ansible as root.
- Store secrets in vault-encrypted files, never in plaintext in the repo.
- Use `--check --diff` for dry-run before applying changes to production.
- Prefer modules (`copy`, `template`, `systemd`) over `shell`/`command` — they are idempotent by design.
- Tag tasks with `tags:` to allow selective execution (`--tags deploy`, `--tags config`).
