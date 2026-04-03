# Data Fetching Skill Guide

## TanStack Query (React Query)

### Setup

```tsx
// main.tsx
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 60_000,        // 1 minute
      retry: 2,
      refetchOnWindowFocus: false,
    },
  },
})

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <Router />
    </QueryClientProvider>
  )
}
```

### useQuery

```ts
import { useQuery } from '@tanstack/react-query'

function useProducts(category?: string) {
  return useQuery({
    queryKey: ['products', { category }],   // array key for granular invalidation
    queryFn: async ({ signal }) => {
      const url = category ? `/api/products?category=${category}` : '/api/products'
      const res = await fetch(url, { signal })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      return res.json() as Promise<Product[]>
    },
    enabled: true,          // conditional: enabled: !!userId
    staleTime: 30_000,
    select: (data) => data.sort((a, b) => a.name.localeCompare(b.name)),
  })
}

// Component
function ProductList() {
  const { data, isLoading, isError, error, refetch } = useProducts('electronics')
  if (isLoading) return <Spinner />
  if (isError) return <p>Error: {error.message}</p>
  return <ul>{data?.map(p => <li key={p.id}>{p.name}</li>)}</ul>
}
```

### useMutation + Cache Invalidation

```ts
import { useMutation, useQueryClient } from '@tanstack/react-query'

function useCreateProduct() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (product: NewProduct) => {
      const res = await fetch('/api/products', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(product),
      })
      if (!res.ok) throw new Error('Failed to create product')
      return res.json() as Promise<Product>
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['products'] })
    },
    onError: (error) => {
      console.error('Mutation failed:', error.message)
    },
  })
}
```

### Optimistic Updates

```ts
function useUpdateProduct() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (update: { id: string; name: string }) =>
      fetch(`/api/products/${update.id}`, {
        method: 'PATCH',
        body: JSON.stringify({ name: update.name }),
      }).then(r => r.json()),

    onMutate: async (update) => {
      await queryClient.cancelQueries({ queryKey: ['products'] })
      const previousProducts = queryClient.getQueryData<Product[]>(['products'])

      queryClient.setQueryData<Product[]>(['products'], (old = []) =>
        old.map(p => p.id === update.id ? { ...p, name: update.name } : p)
      )

      return { previousProducts }
    },

    onError: (err, update, context) => {
      queryClient.setQueryData(['products'], context?.previousProducts)
    },

    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ['products'] })
    },
  })
}
```

---

## Apollo Client

### Setup

```tsx
import { ApolloClient, InMemoryCache, ApolloProvider } from '@apollo/client'

const client = new ApolloClient({
  uri: '/graphql',
  cache: new InMemoryCache(),
  defaultOptions: {
    watchQuery: { fetchPolicy: 'cache-and-network' },
  },
})

export function App() {
  return (
    <ApolloProvider client={client}>
      <Router />
    </ApolloProvider>
  )
}
```

### useQuery / useMutation / useSubscription

```ts
import { gql, useQuery, useMutation, useSubscription } from '@apollo/client'

const GET_PRODUCTS = gql`
  query GetProducts($category: String) {
    products(category: $category) { id name price }
  }
`

const CREATE_PRODUCT = gql`
  mutation CreateProduct($input: ProductInput!) {
    createProduct(input: $input) { id name price }
  }
`

const PRODUCT_UPDATED = gql`
  subscription ProductUpdated($id: ID!) {
    productUpdated(id: $id) { id name price }
  }
`

function ProductList() {
  const { data, loading, error } = useQuery(GET_PRODUCTS, {
    variables: { category: 'electronics' },
    fetchPolicy: 'cache-and-network',
  })

  const [createProduct, { loading: creating }] = useMutation(CREATE_PRODUCT, {
    refetchQueries: [{ query: GET_PRODUCTS }],
  })

  const { data: liveData } = useSubscription(PRODUCT_UPDATED, {
    variables: { id: 'product-1' },
  })

  if (loading) return <Spinner />
  if (error) return <p>Error: {error.message}</p>
  return <ul>{data.products.map((p: Product) => <li key={p.id}>{p.name}</li>)}</ul>
}
```

---

## SWR

```tsx
import useSWR, { SWRConfig, mutate } from 'swr'

const fetcher = (url: string) => fetch(url).then(r => {
  if (!r.ok) throw new Error(`HTTP ${r.status}`)
  return r.json()
})

// Global config
export function App() {
  return (
    <SWRConfig value={{ fetcher, revalidateOnFocus: false, errorRetryCount: 2 }}>
      <Router />
    </SWRConfig>
  )
}

// Usage
function useUser(id: string) {
  const { data, error, isLoading, mutate } = useSWR<User>(
    id ? `/api/users/${id}` : null,   // null disables the fetch
    { refreshInterval: 30_000 }
  )
  return { user: data, error, isLoading, mutate }
}

// Optimistic update
async function updateUsername(id: string, name: string) {
  await mutate(
    `/api/users/${id}`,
    async (current: User) => {
      await fetch(`/api/users/${id}`, {
        method: 'PATCH',
        body: JSON.stringify({ name }),
      })
      return { ...current, name }    // return new object
    },
    { optimisticData: (current: User) => ({ ...current, name }), rollbackOnError: true }
  )
}
```

---

## tRPC

### Server Setup

```ts
// server/router.ts
import { initTRPC } from '@trpc/server'
import { z } from 'zod'

const t = initTRPC.create()

export const appRouter = t.router({
  products: t.router({
    list: t.procedure
      .input(z.object({ category: z.string().optional() }).optional())
      .query(async ({ input }) => {
        return db.product.findMany({ where: { category: input?.category } })
      }),
    create: t.procedure
      .input(z.object({ name: z.string(), price: z.number() }))
      .mutation(async ({ input }) => {
        return db.product.create({ data: input })
      }),
  }),
})

export type AppRouter = typeof appRouter
```

### Client Setup

```ts
// utils/trpc.ts
import { createTRPCReact } from '@trpc/react-query'
import type { AppRouter } from '../server/router'

export const trpc = createTRPCReact<AppRouter>()
```

```tsx
// App.tsx
import { trpc } from './utils/trpc'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { httpBatchLink } from '@trpc/client'

const queryClient = new QueryClient()
const trpcClient = trpc.createClient({
  links: [httpBatchLink({ url: '/api/trpc' })],
})

export function App() {
  return (
    <trpc.Provider client={trpcClient} queryClient={queryClient}>
      <QueryClientProvider client={queryClient}>
        <Router />
      </QueryClientProvider>
    </trpc.Provider>
  )
}
```

```tsx
// Component — fully type-inferred
function ProductList() {
  const { data, isLoading } = trpc.products.list.useQuery({ category: 'electronics' })
  const create = trpc.products.create.useMutation({
    onSuccess: () => trpc.useUtils().products.list.invalidate(),
  })

  if (isLoading) return <Spinner />
  return (
    <div>
      <ul>{data?.map(p => <li key={p.id}>{p.name}</li>)}</ul>
      <button onClick={() => create.mutate({ name: 'Widget', price: 9.99 })}>
        Add Product
      </button>
    </div>
  )
}
```

---

## Key Rules

- Always pass `signal` to `fetch()` inside query functions so React Query can abort stale requests.
- Use array query keys with arguments for granular cache invalidation.
- Optimistic updates: always cancel in-flight queries in `onMutate`, rollback in `onError`.
- SWR: pass `null` as the key to conditionally disable fetching.
- tRPC: export `AppRouter` type from the server and import it on the client — never import server implementations.
- Apollo: prefer `cache-and-network` for lists, `cache-first` for detail views.
