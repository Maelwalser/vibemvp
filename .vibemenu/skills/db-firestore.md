# Firestore Skill Guide

## Overview

Firestore is Google Cloud's serverless document database with real-time sync, offline support, and a hierarchical collection/document/subcollection model. It scales automatically with no index management for simple queries.

## Setup & Connection

```javascript
// Firebase Admin SDK (server-side)
import { initializeApp, cert } from 'firebase-admin/app';
import { getFirestore, FieldValue, Timestamp } from 'firebase-admin/firestore';

initializeApp({ credential: cert(JSON.parse(process.env.FIREBASE_SERVICE_ACCOUNT)) });
const db = getFirestore();

// Firebase Client SDK (browser/mobile)
import { initializeApp } from 'firebase/app';
import { getFirestore, collection, doc, onSnapshot, query, where, orderBy } from 'firebase/firestore';

const app = initializeApp({ projectId: process.env.VITE_FIREBASE_PROJECT_ID, ...config });
const db = getFirestore(app);
```

## Collection / Document / Subcollection Hierarchy

```
/users/{userId}                    ← top-level collection
  /users/{userId}/orders/{orderId} ← subcollection (not nested object)
  /users/{userId}/posts/{postId}   ← another subcollection

/products/{productId}              ← top-level collection
```

```javascript
// Write a document
await db.collection('users').doc(userId).set({
  name: 'Alice',
  email: 'alice@example.com',
  role: 'user',
  createdAt: FieldValue.serverTimestamp(),
});

// Update specific fields (non-destructive)
await db.collection('users').doc(userId).update({
  'profile.bio': 'Developer',
  updatedAt: FieldValue.serverTimestamp(),
});

// Read a document
const snap = await db.collection('users').doc(userId).get();
if (snap.exists) {
  const user = snap.data();
}
```

## Real-time Listeners (onSnapshot)

```javascript
// Client SDK — listen to a document
const unsubscribe = onSnapshot(
  doc(db, 'orders', orderId),
  (snapshot) => {
    if (snapshot.exists()) {
      const order = { id: snapshot.id, ...snapshot.data() };
      renderOrder(order);
    }
  },
  (error) => console.error('Listen error:', error)
);

// Listen to a query
const q = query(
  collection(db, 'orders'),
  where('userId', '==', currentUserId),
  where('status', '==', 'pending'),
  orderBy('createdAt', 'desc')
);
const unsubscribe = onSnapshot(q, (snapshot) => {
  const orders = snapshot.docs.map(d => ({ id: d.id, ...d.data() }));
  renderOrders(orders);
});

// Always clean up listeners
onUnmount(() => unsubscribe());
```

## Security Rules

```javascript
// firestore.rules
rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {

    // Users can read/write their own profile
    match /users/{userId} {
      allow read, write: if request.auth != null && request.auth.uid == userId;
    }

    // Orders: owner can read; only backend (admin SDK) can write
    match /users/{userId}/orders/{orderId} {
      allow read: if request.auth != null && request.auth.uid == userId;
      allow write: if false;  // admin SDK bypasses rules
    }

    // Products: public read, admin write
    match /products/{productId} {
      allow read: if true;
      allow write: if request.auth != null
        && get(/databases/$(database)/documents/users/$(request.auth.uid)).data.role == 'admin';
    }

    // Validate new document structure
    match /orders/{orderId} {
      allow create: if request.auth != null
        && request.resource.data.keys().hasAll(['userId', 'items', 'total'])
        && request.resource.data.total is number
        && request.resource.data.total > 0;
    }
  }
}
```

## Composite Indexes for Multi-Field Queries

```javascript
// This query requires a composite index: status ASC + createdAt DESC
const q = query(
  collection(db, 'orders'),
  where('status', '==', 'completed'),
  orderBy('createdAt', 'desc'),
  limit(20)
);

// Create via Firebase Console or firestore.indexes.json:
// {
//   "indexes": [{
//     "collectionGroup": "orders",
//     "queryScope": "COLLECTION",
//     "fields": [
//       { "fieldPath": "status", "order": "ASCENDING" },
//       { "fieldPath": "createdAt", "order": "DESCENDING" }
//     ]
//   }]
// }
```

## Batch Writes

```javascript
// Atomic batch — up to 500 operations
const batch = db.batch();

batch.set(db.collection('users').doc('alice'), { name: 'Alice' });
batch.update(db.collection('products').doc('prod-1'), { stock: FieldValue.increment(-1) });
batch.delete(db.collection('carts').doc('cart-alice'));

await batch.commit();
```

## Transactions

```javascript
// Read-then-write atomically
await db.runTransaction(async (txn) => {
  const productRef = db.collection('products').doc(productId);
  const product = await txn.get(productRef);

  if (!product.exists) throw new Error('Product not found');
  if (product.data().stock < qty) throw new Error('Insufficient stock');

  txn.update(productRef, { stock: FieldValue.increment(-qty) });
  txn.set(db.collection('orders').doc(), {
    userId,
    productId,
    qty,
    total: product.data().price * qty,
    createdAt: FieldValue.serverTimestamp(),
  });
});
```

## collectionGroup Queries

```javascript
// Query across ALL subcollections named "orders" (regardless of parent)
// Requires a single-field or composite index with "Collection group" scope
const allOrders = query(
  collectionGroup(db, 'orders'),
  where('status', '==', 'pending'),
  orderBy('createdAt', 'asc')
);

const snapshot = await getDocs(allOrders);
```

## Cursor Pagination

```javascript
// Keyset pagination using startAfter
let lastDoc = null;

async function loadNextPage() {
  let q = query(
    collection(db, 'products'),
    orderBy('createdAt', 'desc'),
    limit(20)
  );
  if (lastDoc) q = query(q, startAfter(lastDoc));

  const snapshot = await getDocs(q);
  lastDoc = snapshot.docs[snapshot.docs.length - 1];
  return snapshot.docs.map(d => ({ id: d.id, ...d.data() }));
}
```

## Key Rules

- Firestore charges per document read/write — minimize reads with projections (admin SDK supports `select()`)
- Subcollections are not deleted when parent document is deleted — delete them explicitly
- Maximum document size is 1 MiB — store large payloads in Cloud Storage
- Security rules apply only to client SDK — admin SDK bypasses them entirely
- Composite indexes must be created before running multi-field queries
- `collectionGroup` queries require an index with "Collection group" scope
- Transactions automatically retry on contention — keep them short and idempotent
- `FieldValue.serverTimestamp()` is preferred over `new Date()` to avoid clock skew
