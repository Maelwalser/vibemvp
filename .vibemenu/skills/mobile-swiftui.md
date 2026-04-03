# SwiftUI Skill Guide

## Project Layout

```
App/
├── App.xcodeproj
└── App/
    ├── AppMain.swift           # @main entry point
    ├── AppView.swift           # Root NavigationStack
    ├── Features/
    │   └── Users/
    │       ├── UsersView.swift
    │       ├── UserDetailView.swift
    │       └── UsersViewModel.swift
    ├── Shared/
    │   ├── Components/
    │   │   └── AsyncImageView.swift
    │   ├── Services/
    │   │   └── APIClient.swift
    │   └── Models/
    │       └── User.swift
    └── Preview Content/
        └── Preview Assets.xcassets
```

## View Protocol + Body

```swift
struct UserCard: View {
    let user: User

    var body: some View {
        HStack(spacing: 12) {
            AsyncImage(url: URL(string: user.avatarURL)) { image in
                image.resizable().scaledToFill()
            } placeholder: {
                ProgressView()
            }
            .frame(width: 48, height: 48)
            .clipShape(Circle())

            VStack(alignment: .leading, spacing: 4) {
                Text(user.name)
                    .font(.headline)
                Text(user.email)
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }
            Spacer()
        }
        .padding()
        .background(.background)
        .cornerRadius(12)
    }
}

#Preview {
    UserCard(user: .preview)
        .padding()
}
```

## @Observable (iOS 17+)

```swift
import Observation

@Observable
class UsersViewModel {
    var users: [User] = []
    var isLoading = false
    var errorMessage: String?

    private let service: UserService

    init(service: UserService = UserService()) {
        self.service = service
    }

    func loadUsers() async {
        isLoading = true
        defer { isLoading = false }
        do {
            users = try await service.getUsers()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
```

## @ObservableObject (iOS 16 and earlier)

```swift
final class UsersViewModelLegacy: ObservableObject {
    @Published var users: [User] = []
    @Published var isLoading = false

    func loadUsers() async {
        await MainActor.run { isLoading = true }
        let result = try? await UserService().getUsers()
        await MainActor.run {
            users = result ?? []
            isLoading = false
        }
    }
}

// Usage in View
@StateObject private var viewModel = UsersViewModelLegacy()
```

## @State, @Binding, @Environment

```swift
struct CounterView: View {
    // Local state — owned by this view
    @State private var count = 0

    var body: some View {
        VStack {
            Text("Count: \(count)")
            StepperButton(value: $count)   // pass Binding
        }
    }
}

struct StepperButton: View {
    @Binding var value: Int    // child reads + writes parent's state

    var body: some View {
        HStack {
            Button("-") { value -= 1 }
            Button("+") { value += 1 }
        }
    }
}

// @Environment for DI
struct ContentView: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(AuthManager.self) private var auth   // @Observable injected

    var body: some View {
        Text(colorScheme == .dark ? "Dark" : "Light")
    }
}

// Inject into environment
ContentView()
    .environment(authManager)   // @Observable class instance
```

## NavigationStack + navigationDestination

```swift
struct AppView: View {
    @State private var path = NavigationPath()

    var body: some View {
        NavigationStack(path: $path) {
            UserListView()
                .navigationTitle("Users")
                .navigationDestination(for: User.self) { user in
                    UserDetailView(user: user)
                }
                .navigationDestination(for: String.self) { route in
                    if route == "settings" { SettingsView() }
                }
        }
    }
}

// Navigate programmatically
path.append(user)          // push UserDetailView
path.append("settings")    // push SettingsView
path.removeLast()          // pop
path = NavigationPath()    // pop to root
```

## LazyVStack / LazyHStack

```swift
ScrollView {
    LazyVStack(spacing: 12) {
        ForEach(users) { user in
            UserCard(user: user)
        }
    }
    .padding()
}

// List (preferred for long lists — built-in cell recycling)
List(users) { user in
    NavigationLink(value: user) {
        UserCard(user: user)
    }
}
.listStyle(.plain)
```

## @ViewBuilder (Custom Containers)

```swift
struct Card<Content: View>: View {
    let title: String
    @ViewBuilder let content: () -> Content

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(title).font(.headline)
            content()
        }
        .padding()
        .background(Color(.secondarySystemBackground))
        .cornerRadius(12)
    }
}

// Usage
Card(title: "Stats") {
    Text("Users: 42")
    Text("Active: 35")
}
```

## Async Image + Task

```swift
struct UserDetailView: View {
    let userId: String
    @State private var user: User?
    @State private var error: Error?

    var body: some View {
        Group {
            if let user {
                UserProfile(user: user)
            } else if error != nil {
                ContentUnavailableView("Failed to load", systemImage: "xmark.circle")
            } else {
                ProgressView()
            }
        }
        .task {
            do {
                user = try await UserService().getUser(id: userId)
            } catch {
                self.error = error
            }
        }
    }
}
```

## Compared to UIKit

| SwiftUI | UIKit |
|---------|-------|
| Declarative — describe what to show | Imperative — describe how to mutate |
| `@State` / `@Observable` | Override lifecycle methods |
| `NavigationStack` | `UINavigationController` |
| `List` | `UITableView` + dataSource + delegate |
| `.onAppear` | `viewWillAppear` |
| `@Environment` | Dependency injection via init / service locator |
| `#Preview` macro | Storyboard / XIB previews |

## Key Rules

- Use `@Observable` (iOS 17+); fall back to `@ObservableObject` + `@Published` for iOS 16.
- Use `@State` for local view state; hoist to a ViewModel for shared/complex state.
- Pass `@Binding` to child views that need to write back to parent state.
- Use `NavigationStack` (not deprecated `NavigationView`) for all navigation.
- Always use `LazyVStack`/`List` for large datasets — `VStack` renders all items eagerly.
- Use `.task { }` modifier for async work tied to view lifecycle (auto-cancelled on disappear).
- Use `#Preview` macro (Xcode 15+) for all previews — no `PreviewProvider` boilerplate.
- Inject shared objects via `.environment(obj)` and read with `@Environment(Type.self)`.
