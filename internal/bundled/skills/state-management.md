# State Management Skill Guide

## Zustand

### Basic Store

```ts
import { create } from 'zustand'

interface CounterState {
  count: number
  increment: () => void
  decrement: () => void
  reset: () => void
}

export const useCounterStore = create<CounterState>((set) => ({
  count: 0,
  increment: () => set((state) => ({ count: state.count + 1 })),
  decrement: () => set((state) => ({ count: state.count - 1 })),
  reset: () => set({ count: 0 }),
}))

// Usage
function Counter() {
  const { count, increment } = useCounterStore()
  return <button onClick={increment}>{count}</button>
}
```

### Slices Pattern (Multiple Stores)

```ts
import { create } from 'zustand'

// auth slice
const createAuthSlice = (set: any) => ({
  user: null as User | null,
  token: null as string | null,
  login: (user: User, token: string) => set({ user, token }),
  logout: () => set({ user: null, token: null }),
})

// cart slice
const createCartSlice = (set: any) => ({
  items: [] as CartItem[],
  addItem: (item: CartItem) =>
    set((state: any) => ({ items: [...state.items, item] })),
  removeItem: (id: string) =>
    set((state: any) => ({ items: state.items.filter((i: CartItem) => i.id !== id) })),
})

export const useStore = create<ReturnType<typeof createAuthSlice> & ReturnType<typeof createCartSlice>>()(
  (...a) => ({
    ...createAuthSlice(...a),
    ...createCartSlice(...a),
  })
)
```

### Middleware: devtools + persist + subscribeWithSelector

```ts
import { create } from 'zustand'
import { devtools, persist, subscribeWithSelector } from 'zustand/middleware'

export const useSettingsStore = create<SettingsState>()(
  devtools(
    persist(
      subscribeWithSelector((set) => ({
        theme: 'light' as 'light' | 'dark',
        language: 'en',
        setTheme: (theme) => set({ theme }),
        setLanguage: (language) => set({ language }),
      })),
      {
        name: 'settings-storage',           // localStorage key
        partialize: (state) => ({ theme: state.theme }),  // only persist theme
      }
    ),
    { name: 'SettingsStore' }
  )
)

// Subscribe to specific slice without triggering re-render
useSettingsStore.subscribe(
  (state) => state.theme,
  (theme) => document.documentElement.classList.toggle('dark', theme === 'dark')
)
```

---

## Redux Toolkit

### createSlice

```ts
import { createSlice, PayloadAction } from '@reduxjs/toolkit'

interface ProductsState {
  items: Product[]
  status: 'idle' | 'loading' | 'succeeded' | 'failed'
  error: string | null
}

const productsSlice = createSlice({
  name: 'products',
  initialState: { items: [], status: 'idle', error: null } as ProductsState,
  reducers: {
    addProduct: (state, action: PayloadAction<Product>) => {
      state.items.push(action.payload)          // Immer handles immutability
    },
    removeProduct: (state, action: PayloadAction<string>) => {
      state.items = state.items.filter(p => p.id !== action.payload)
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchProducts.pending, (state) => {
        state.status = 'loading'
      })
      .addCase(fetchProducts.fulfilled, (state, action) => {
        state.status = 'succeeded'
        state.items = action.payload
      })
      .addCase(fetchProducts.rejected, (state, action) => {
        state.status = 'failed'
        state.error = action.error.message ?? 'Unknown error'
      })
  },
})

export const { addProduct, removeProduct } = productsSlice.actions
export default productsSlice.reducer
```

### createAsyncThunk

```ts
import { createAsyncThunk } from '@reduxjs/toolkit'

export const fetchProducts = createAsyncThunk(
  'products/fetchAll',
  async (_, { rejectWithValue }) => {
    try {
      const res = await fetch('/api/products')
      if (!res.ok) throw new Error('Server error')
      return (await res.json()) as Product[]
    } catch (err) {
      return rejectWithValue((err as Error).message)
    }
  }
)
```

### RTK Query

```ts
import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react'

export const productsApi = createApi({
  reducerPath: 'productsApi',
  baseQuery: fetchBaseQuery({ baseUrl: '/api' }),
  tagTypes: ['Product'],
  endpoints: (builder) => ({
    getProducts: builder.query<Product[], void>({
      query: () => '/products',
      providesTags: ['Product'],
    }),
    getProductById: builder.query<Product, string>({
      query: (id) => `/products/${id}`,
      providesTags: (result, error, id) => [{ type: 'Product', id }],
    }),
    createProduct: builder.mutation<Product, Partial<Product>>({
      query: (body) => ({ url: '/products', method: 'POST', body }),
      invalidatesTags: ['Product'],
    }),
  }),
})

export const { useGetProductsQuery, useGetProductByIdQuery, useCreateProductMutation } = productsApi
```

---

## Jotai

```ts
import { atom, useAtom } from 'jotai'
import { atomWithStorage } from 'jotai/utils'

// Primitive atom
const countAtom = atom(0)

// Derived (read-only) atom
const doubledAtom = atom((get) => get(countAtom) * 2)

// Writable derived atom
const countStringAtom = atom(
  (get) => String(get(countAtom)),
  (get, set, value: string) => set(countAtom, Number(value))
)

// Persistent atom (uses localStorage)
const themeAtom = atomWithStorage<'light' | 'dark'>('theme', 'light')

// Async atom
const userAtom = atom(async () => {
  const res = await fetch('/api/me')
  return res.json() as Promise<User>
})

// Usage
function Counter() {
  const [count, setCount] = useAtom(countAtom)
  const [doubled] = useAtom(doubledAtom)
  return (
    <div>
      <span>{count} (doubled: {doubled})</span>
      <button onClick={() => setCount(c => c + 1)}>+</button>
    </div>
  )
}
```

---

## Pinia (Vue)

```ts
// stores/user.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

// Composition-style store (recommended)
export const useUserStore = defineStore('user', () => {
  const user = ref<User | null>(null)
  const token = ref<string | null>(null)

  const isAuthenticated = computed(() => !!user.value)
  const displayName = computed(() => user.value?.name ?? 'Guest')

  async function login(credentials: { email: string; password: string }) {
    const res = await fetch('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify(credentials),
    })
    const data = await res.json()
    user.value = data.user
    token.value = data.token
  }

  function logout() {
    user.value = null
    token.value = null
  }

  return { user, token, isAuthenticated, displayName, login, logout }
})
```

```vue
<!-- Component usage -->
<script setup>
import { storeToRefs } from 'pinia'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()
const { user, isAuthenticated } = storeToRefs(userStore)  // keep reactivity
</script>

<template>
  <span v-if="isAuthenticated">{{ user.name }}</span>
  <button @click="userStore.logout()">Logout</button>
</template>
```

---

## Svelte Stores

```ts
// stores/cart.ts
import { writable, readable, derived, get } from 'svelte/store'

// Writable store
export const cart = writable<CartItem[]>([])

// Derived store
export const cartTotal = derived(
  cart,
  ($cart) => $cart.reduce((sum, item) => sum + item.price * item.qty, 0)
)

// Readable store (external data source)
export const time = readable(new Date(), (set) => {
  const interval = setInterval(() => set(new Date()), 1000)
  return () => clearInterval(interval)
})

// Store actions (mutate immutably)
export function addToCart(item: CartItem) {
  cart.update((items) => {
    const existing = items.find((i) => i.id === item.id)
    if (existing) {
      return items.map((i) => i.id === item.id ? { ...i, qty: i.qty + 1 } : i)
    }
    return [...items, { ...item, qty: 1 }]
  })
}

export function removeFromCart(id: string) {
  cart.update((items) => items.filter((i) => i.id !== id))
}
```

```svelte
<!-- Component: $store auto-subscribes and unsubscribes -->
<script>
  import { cart, cartTotal, addToCart } from '$lib/stores/cart'
</script>

<p>Items: {$cart.length} — Total: ${$cartTotal.toFixed(2)}</p>
```

---

## Angular Signals

```ts
import { Component, signal, computed, effect } from '@angular/core'
import { toObservable, toSignal } from '@angular/core/rxjs-interop'
import { interval } from 'rxjs'
import { map } from 'rxjs/operators'

@Component({
  selector: 'app-counter',
  template: `
    <p>Count: {{ count() }} | Doubled: {{ doubled() }}</p>
    <button (click)="increment()">+</button>
  `,
})
export class CounterComponent {
  count = signal(0)
  doubled = computed(() => this.count() * 2)

  constructor() {
    // Effect runs whenever count changes
    effect(() => {
      console.log('Count changed:', this.count())
    })
  }

  increment() {
    this.count.update((c) => c + 1)
  }
}

// RxJS interop
class DataComponent {
  // Signal → Observable
  count = signal(0)
  count$ = toObservable(this.count)

  // Observable → Signal
  ticker$ = interval(1000).pipe(map((n) => `Tick ${n}`))
  ticker = toSignal(this.ticker$, { initialValue: 'Tick 0' })
}
```

---

## Key Rules

- Zustand: prefer one store per domain; use `subscribeWithSelector` for side effects.
- Redux Toolkit: use RTK Query instead of manual `createAsyncThunk` for data fetching.
- Jotai: fine-grained atoms beat one big context; use `atomWithStorage` for persistence.
- Pinia: always use `storeToRefs()` when destructuring to preserve reactivity.
- Svelte: update stores via `.update(fn)` returning a new array/object, never mutate directly.
- Angular: prefer Signals over `BehaviorSubject` for local component state.
