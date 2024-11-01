import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'record.dart';

void main() => runApp(const MyApp());

/// The route configuration.
final GoRouter _router = GoRouter(
  routes: <RouteBase>[
    GoRoute(
      path: '/',
      builder: (BuildContext context, GoRouterState state) {
        return const HomeScreen();
      },
      routes: <RouteBase>[
        GoRoute(
          path: 'record',
          builder: (BuildContext context, GoRouterState state) {
            return const RecordScreen();
          },
        ),
      ],
    ),
  ],
);

/// The main app.
class MyApp extends StatelessWidget {
  /// Constructs a [MyApp]
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      routerConfig: _router,
    );
  }
}

/// The home screen
class HomeScreen extends StatelessWidget {
  /// Constructs a [HomeScreen]
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Home Screen')),
      body: Center(
        child: ElevatedButton(
          onPressed: () => context.go('/record'),
          child: const Text('Go to the Record screen'),
        ),
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: () => context.go('/record'),
        child: const Icon(Icons.receipt),
      ),
    );
  }
}
