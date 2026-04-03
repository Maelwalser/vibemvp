# UI Component Libraries Skill Guide

## shadcn/ui

### Setup

```bash
npx shadcn-ui@latest init
npx shadcn-ui@latest add button card input dialog table
```

Components are copied into `components/ui/` — they are your code, fully customizable.

### Usage

```typescript
import { Button } from '@/components/ui/button';
import { Card, CardHeader, CardTitle, CardContent, CardFooter } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

function UserForm() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Create User</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="name">Name</Label>
          <Input id="name" placeholder="Alice" />
        </div>
      </CardContent>
      <CardFooter>
        <Button type="submit" className="w-full">Create</Button>
        <Button variant="outline">Cancel</Button>
      </CardFooter>
    </Card>
  );
}
```

### CSS Variable Theming

```css
/* globals.css — customize these to change the entire theme */
:root {
  --background: 0 0% 100%;
  --foreground: 222.2 84% 4.9%;
  --primary: 221.2 83.2% 53.3%;
  --primary-foreground: 210 40% 98%;
  --radius: 0.5rem;
}
.dark {
  --background: 222.2 84% 4.9%;
  --primary: 217.2 91.2% 59.8%;
}
```

---

## Material UI (MUI)

### Setup

```bash
npm install @mui/material @emotion/react @emotion/styled @mui/icons-material
```

### ThemeProvider

```typescript
import { createTheme, ThemeProvider, CssBaseline } from '@mui/material';

const theme = createTheme({
  palette: {
    mode: 'dark',
    primary: { main: '#6366f1' },
    secondary: { main: '#ec4899' },
  },
  shape: { borderRadius: 12 },
  typography: {
    fontFamily: '"Inter", "Roboto", sans-serif',
  },
});

function App() {
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <YourApp />
    </ThemeProvider>
  );
}
```

### sx Prop

```typescript
import { Box, Stack, Typography, Button } from '@mui/material';

function UserCard({ user }: { user: User }) {
  return (
    <Box
      sx={{
        p: 2,                        // padding: 16px (theme.spacing(2))
        borderRadius: 2,
        bgcolor: 'background.paper',
        border: '1px solid',
        borderColor: 'divider',
        '&:hover': {                 // pseudo-class
          borderColor: 'primary.main',
          boxShadow: 3,
        },
        transition: 'all 0.2s',
      }}
    >
      <Stack direction="row" spacing={2} alignItems="center">
        <Typography variant="h6">{user.name}</Typography>
        <Button variant="contained" size="small" sx={{ ml: 'auto' }}>
          Edit
        </Button>
      </Stack>
      <Typography variant="body2" color="text.secondary">{user.email}</Typography>
    </Box>
  );
}
```

### Grid2

```typescript
import Grid from '@mui/material/Grid2';

<Grid container spacing={2}>
  <Grid size={{ xs: 12, md: 6 }}><Widget /></Grid>
  <Grid size={{ xs: 12, md: 6 }}><Widget /></Grid>
</Grid>
```

---

## Ant Design

### Setup

```bash
npm install antd @ant-design/icons
```

### Form with Validation

```typescript
import { Form, Input, Button, Select, message } from 'antd';

const { Option } = Select;

function CreateUserForm() {
  const [form] = Form.useForm();

  const onFinish = async (values: { name: string; role: string }) => {
    try {
      await createUser(values);
      message.success('User created');
      form.resetFields();
    } catch {
      message.error('Failed to create user');
    }
  };

  return (
    <Form form={form} layout="vertical" onFinish={onFinish}>
      <Form.Item name="name" label="Name" rules={[{ required: true, min: 2 }]}>
        <Input placeholder="Alice" />
      </Form.Item>
      <Form.Item name="role" label="Role" rules={[{ required: true }]}>
        <Select placeholder="Select role">
          <Option value="admin">Admin</Option>
          <Option value="user">User</Option>
        </Select>
      </Form.Item>
      <Form.Item>
        <Button type="primary" htmlType="submit">Create</Button>
      </Form.Item>
    </Form>
  );
}
```

### Table

```typescript
import { Table, type ColumnsType } from 'antd';

const columns: ColumnsType<User> = [
  { title: 'Name', dataIndex: 'name', sorter: (a, b) => a.name.localeCompare(b.name) },
  { title: 'Email', dataIndex: 'email' },
  {
    title: 'Actions',
    render: (_, record) => (
      <Button danger onClick={() => deleteUser(record.id)}>Delete</Button>
    ),
  },
];

<Table
  dataSource={users}
  columns={columns}
  rowKey="id"
  pagination={{ pageSize: 20 }}
  loading={loading}
/>
```

---

## Radix UI (Primitives)

### Setup

```bash
npm install @radix-ui/react-dialog @radix-ui/react-dropdown-menu @radix-ui/react-tabs
```

### Dialog Pattern

```typescript
import * as Dialog from '@radix-ui/react-dialog';

function UserDialog({ user, children }: { user: User; children: React.ReactNode }) {
  return (
    <Dialog.Root>
      <Dialog.Trigger asChild>{children}</Dialog.Trigger>
      <Dialog.Portal>
        <Dialog.Overlay className="dialog-overlay" />
        <Dialog.Content className="dialog-content">
          <Dialog.Title>Edit {user.name}</Dialog.Title>
          <Dialog.Description>Update user details.</Dialog.Description>

          {/* Form content */}
          <input defaultValue={user.name} />

          <Dialog.Close asChild>
            <button className="close-button">Cancel</button>
          </Dialog.Close>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

// Usage
<UserDialog user={user}>
  <button>Edit User</button>
</UserDialog>
```

### Dropdown Menu

```typescript
import * as DropdownMenu from '@radix-ui/react-dropdown-menu';

function ActionMenu({ onEdit, onDelete }: Props) {
  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <button>Actions</button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content className="dropdown-content" sideOffset={4}>
          <DropdownMenu.Item onSelect={onEdit}>Edit</DropdownMenu.Item>
          <DropdownMenu.Separator />
          <DropdownMenu.Item onSelect={onDelete} className="destructive">
            Delete
          </DropdownMenu.Item>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}
```

### Tabs

```typescript
import * as Tabs from '@radix-ui/react-tabs';

<Tabs.Root defaultValue="profile">
  <Tabs.List>
    <Tabs.Trigger value="profile">Profile</Tabs.Trigger>
    <Tabs.Trigger value="settings">Settings</Tabs.Trigger>
  </Tabs.List>
  <Tabs.Content value="profile"><ProfileTab /></Tabs.Content>
  <Tabs.Content value="settings"><SettingsTab /></Tabs.Content>
</Tabs.Root>
```

---

## Headless UI

### Setup

```bash
npm install @headlessui/react
```

### Listbox (Select)

```typescript
import { Listbox } from '@headlessui/react';

function RoleSelect({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const roles = ['admin', 'editor', 'viewer'];
  return (
    <Listbox value={value} onChange={onChange}>
      <Listbox.Button className="select-button">{value}</Listbox.Button>
      <Listbox.Options className="select-options">
        {roles.map(role => (
          <Listbox.Option key={role} value={role}>
            {({ active, selected }) => (
              <li className={`option ${active ? 'active' : ''} ${selected ? 'selected' : ''}`}>
                {role}
              </li>
            )}
          </Listbox.Option>
        ))}
      </Listbox.Options>
    </Listbox>
  );
}
```

### Switch (Toggle)

```typescript
import { Switch } from '@headlessui/react';

function Toggle({ enabled, onToggle }: { enabled: boolean; onToggle: () => void }) {
  return (
    <Switch
      checked={enabled}
      onChange={onToggle}
      className={`toggle ${enabled ? 'enabled' : ''}`}
    >
      <span className="toggle-thumb" />
    </Switch>
  );
}
```

---

## DaisyUI

### Setup

```bash
npm install daisyui
# tailwind.config.js:  plugins: [require('daisyui')]
```

### Component Classes

```html
<!-- Buttons -->
<button class="btn btn-primary">Primary</button>
<button class="btn btn-secondary btn-outline">Outline</button>
<button class="btn btn-error btn-sm">Delete</button>

<!-- Card -->
<div class="card bg-base-100 shadow-xl w-96">
  <div class="card-body">
    <h2 class="card-title">Card Title</h2>
    <p>Card content goes here.</p>
    <div class="card-actions justify-end">
      <button class="btn btn-primary">Action</button>
    </div>
  </div>
</div>

<!-- Form -->
<label class="form-control w-full">
  <div class="label"><span class="label-text">Name</span></div>
  <input type="text" class="input input-bordered" placeholder="Alice" />
</label>

<!-- Badge -->
<span class="badge badge-primary">New</span>
<span class="badge badge-error badge-outline">Error</span>

<!-- Alert -->
<div role="alert" class="alert alert-success">
  <span>User created successfully!</span>
</div>

<!-- Modal -->
<dialog id="confirm_modal" class="modal">
  <div class="modal-box">
    <h3 class="font-bold text-lg">Confirm</h3>
    <p>Are you sure?</p>
    <div class="modal-action">
      <form method="dialog">
        <button class="btn">Cancel</button>
        <button class="btn btn-error">Delete</button>
      </form>
    </div>
  </div>
</dialog>
<button onclick="confirm_modal.showModal()" class="btn">Open</button>
```

### Theming

```html
<!-- Apply theme to entire page -->
<html data-theme="dark">
<!-- Or switch dynamically -->
<html data-theme="cupcake">
```

```javascript
// Switch theme dynamically
document.documentElement.setAttribute('data-theme', 'night');
```

Available themes: `light`, `dark`, `cupcake`, `bumblebee`, `emerald`, `corporate`, `synthwave`, `retro`, `cyberpunk`, `valentine`, `halloween`, `garden`, `forest`, `aqua`, `lofi`, `pastel`, `fantasy`, `wireframe`, `black`, `luxury`, `dracula`, `cmyk`, `autumn`, `business`, `acid`, `lemonade`, `night`, `coffee`, `winter`

## Key Rules

- **shadcn/ui**: Components are copied, not imported — customize freely in `components/ui/`.
- **MUI**: Use `sx` prop for one-off styles; create theme overrides for systematic changes.
- **Ant Design**: Always use `Form.useForm()` hook — never control form state manually.
- **Radix UI**: Always use `Portal` for overlays (Dialog, Dropdown) to escape stacking context.
- **Headless UI**: Render prop pattern provides `active`/`selected`/`open` state — use for conditional classes.
- **DaisyUI**: All components are pure CSS classes — no JS required, compatible with any framework.
