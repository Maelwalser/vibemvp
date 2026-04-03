# Flutter Skill Guide

## Project Layout

```
app/
├── pubspec.yaml
├── lib/
│   ├── main.dart               # Entry point
│   ├── app.dart                # MaterialApp + router
│   ├── core/
│   │   ├── router/
│   │   │   └── app_router.dart # GoRouter config
│   │   └── theme/
│   │       └── app_theme.dart
│   ├── features/
│   │   └── users/
│   │       ├── data/
│   │       │   ├── models/
│   │       │   └── repositories/
│   │       ├── domain/
│   │       │   └── entities/
│   │       ├── presentation/
│   │       │   ├── bloc/
│   │       │   │   ├── users_bloc.dart
│   │       │   │   ├── users_event.dart
│   │       │   │   └── users_state.dart
│   │       │   └── pages/
│   │       │       └── users_page.dart
│   │       └── users_module.dart
│   └── shared/
│       └── widgets/
└── test/
```

## Key Dependencies (pubspec.yaml)

```yaml
dependencies:
  flutter:
    sdk: flutter
  flutter_bloc: ^8.1.6
  go_router: ^14.3.0
  get_it: ^8.0.0
  dio: ^5.7.0
  equatable: ^2.0.7
  freezed_annotation: ^2.4.4

dev_dependencies:
  flutter_test:
    sdk: flutter
  build_runner: ^2.4.13
  freezed: ^2.5.7
  flutter_lints: ^4.0.0
  mocktail: ^1.0.4
```

## StatelessWidget / StatefulWidget

```dart
// StatelessWidget — no mutable state
class UserCard extends StatelessWidget {
  const UserCard({super.key, required this.user});
  final User user;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: ListTile(
        title: Text(user.name),
        subtitle: Text(user.email),
      ),
    );
  }
}

// StatefulWidget — has mutable local state
class CounterWidget extends StatefulWidget {
  const CounterWidget({super.key});
  @override
  State<CounterWidget> createState() => _CounterWidgetState();
}

class _CounterWidgetState extends State<CounterWidget> {
  int _count = 0;

  @override
  Widget build(BuildContext context) {
    return TextButton(
      onPressed: () => setState(() => _count++),
      child: Text('Count: $_count'),
    );
  }
}
```

## BLoC Pattern

```dart
// users_event.dart
abstract class UsersEvent extends Equatable {
  @override List<Object?> get props => [];
}
class LoadUsers extends UsersEvent {}
class DeleteUser extends UsersEvent {
  const DeleteUser(this.id);
  final String id;
  @override List<Object?> get props => [id];
}

// users_state.dart
abstract class UsersState extends Equatable {
  @override List<Object?> get props => [];
}
class UsersInitial extends UsersState {}
class UsersLoading extends UsersState {}
class UsersLoaded extends UsersState {
  const UsersLoaded(this.users);
  final List<User> users;
  @override List<Object?> get props => [users];
}
class UsersError extends UsersState {
  const UsersError(this.message);
  final String message;
  @override List<Object?> get props => [message];
}

// users_bloc.dart
class UsersBloc extends Bloc<UsersEvent, UsersState> {
  UsersBloc(this._repository) : super(UsersInitial()) {
    on<LoadUsers>(_onLoadUsers);
    on<DeleteUser>(_onDeleteUser);
  }

  final UserRepository _repository;

  Future<void> _onLoadUsers(LoadUsers event, Emitter<UsersState> emit) async {
    emit(UsersLoading());
    try {
      final users = await _repository.getUsers();
      emit(UsersLoaded(users));
    } catch (e) {
      emit(UsersError(e.toString()));
    }
  }

  Future<void> _onDeleteUser(DeleteUser event, Emitter<UsersState> emit) async {
    await _repository.deleteUser(event.id);
    add(LoadUsers());   // re-fetch
  }
}
```

## BlocBuilder / BlocListener

```dart
class UsersPage extends StatelessWidget {
  const UsersPage({super.key});

  @override
  Widget build(BuildContext context) {
    return BlocProvider(
      create: (_) => UsersBloc(getIt<UserRepository>())..add(LoadUsers()),
      child: BlocListener<UsersBloc, UsersState>(
        listener: (context, state) {
          if (state is UsersError) {
            ScaffoldMessenger.of(context)
              .showSnackBar(SnackBar(content: Text(state.message)));
          }
        },
        child: BlocBuilder<UsersBloc, UsersState>(
          builder: (context, state) {
            return switch (state) {
              UsersInitial() || UsersLoading() =>
                const Center(child: CircularProgressIndicator()),
              UsersLoaded() => ListView.builder(
                  itemCount: state.users.length,
                  itemBuilder: (_, i) => UserCard(user: state.users[i]),
                ),
              UsersError() => Center(child: Text(state.message)),
            };
          },
        ),
      ),
    );
  }
}
```

## GoRouter

```dart
// lib/core/router/app_router.dart
import 'package:go_router/go_router.dart';

final appRouter = GoRouter(
  initialLocation: '/',
  redirect: (context, state) {
    final isAuth = getIt<AuthService>().isAuthenticated;
    final isLogin = state.matchedLocation == '/login';
    if (!isAuth && !isLogin) return '/login';
    if (isAuth && isLogin) return '/';
    return null;
  },
  routes: [
    GoRoute(path: '/', builder: (_, __) => const HomePage()),
    GoRoute(path: '/login', builder: (_, __) => const LoginPage()),
    GoRoute(
      path: '/users',
      builder: (_, __) => const UsersPage(),
      routes: [
        GoRoute(
          path: ':id',
          builder: (_, state) => UserDetailPage(id: state.pathParameters['id']!),
        ),
      ],
    ),
  ],
);

// Navigation
context.go('/users');
context.push('/users/123');
context.pop();
```

## FutureBuilder / StreamBuilder

```dart
// FutureBuilder
FutureBuilder<User>(
  future: repository.getUser(id),
  builder: (context, snapshot) {
    if (snapshot.connectionState == ConnectionState.waiting) {
      return const CircularProgressIndicator();
    }
    if (snapshot.hasError) return Text('Error: ${snapshot.error}');
    return UserCard(user: snapshot.requireData);
  },
)

// StreamBuilder — live updates
StreamBuilder<List<Message>>(
  stream: chatRepository.messagesStream(roomId),
  builder: (context, snapshot) {
    if (!snapshot.hasData) return const SizedBox.shrink();
    return MessageList(messages: snapshot.requireData);
  },
)
```

## ListView.builder (Performance)

```dart
ListView.builder(
  itemCount: users.length,
  itemBuilder: (context, index) {
    final user = users[index];
    return UserCard(key: ValueKey(user.id), user: user);
  },
)
```

## Theming

```dart
// Inherit theme tokens
final colorScheme = Theme.of(context).colorScheme;
final textTheme = Theme.of(context).textTheme;

Container(
  color: colorScheme.surface,
  child: Text('Hello', style: textTheme.titleLarge),
)
```

## Platform Channels (Native)

```dart
// Dart side
static const _channel = MethodChannel('com.example.app/biometrics');

Future<bool> authenticate() async {
  try {
    return await _channel.invokeMethod<bool>('authenticate') ?? false;
  } on PlatformException catch (e) {
    throw Exception('Biometrics failed: ${e.message}');
  }
}
```

## Key Rules

- Prefer `const` constructors everywhere — Flutter skips rebuilding const widgets.
- Use `BlocProvider` at the route level, not deep in the widget tree.
- Extend `Equatable` in all BLoC events and states for correct equality.
- Use `GoRouter` — avoid Navigator 1.0 push/pop for new code.
- Use `ListView.builder` for any list longer than 20 items.
- Never put business logic in `build()` — use BLoC/Cubit or repository.
- Use `ValueKey(item.id)` in list builders for stable identity.
- All colors and text styles must come from `Theme.of(context)` — no hardcoded values.
