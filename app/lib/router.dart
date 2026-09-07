import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'core/auth/auth_controller.dart';
import 'core/auth/auth_state.dart';
import 'features/activate_account/activate_account_screen.dart';
import 'features/create_user/create_user_screen.dart';
import 'features/forgot_password/forgot_password_screen.dart';
import 'features/home/home_screen.dart';
import 'features/login/login_screen.dart';
import 'features/manage_users/manage_users_screen.dart';
import 'features/profile/profile_screen.dart';
import 'features/reset_password/reset_password_screen.dart';
import 'features/splash/splash_screen.dart';

/// Sostituisce il taglio netto di default fra una schermata e l'altra con
/// una dissolvenza incrociata: piccolo dettaglio, ma è uno di quei tocchi
/// che fanno percepire un'app come curata invece che abbozzata. Usata da
/// ogni `GoRoute` sotto tramite `pageBuilder` invece del più semplice
/// `builder` (che userebbe la transizione di default della piattaforma).
CustomTransitionPage<void> _fadeTransitionPage(Widget child) {
  return CustomTransitionPage<void>(
    child: child,
    transitionDuration: const Duration(milliseconds: 220),
    transitionsBuilder: (context, animation, secondaryAnimation, child) {
      return FadeTransition(opacity: animation, child: child);
    },
  );
}

/// `go_router` costruisce la mappa delle route dell'app: quale schermata
/// mostrare per ogni indirizzo (es. "/login" -> `LoginScreen`), e con
/// `redirect` una regola che gira PRIMA di ogni navigazione e può dirottarla
/// altrove — qui la usiamo per proteggere "/profilo": chi non ha fatto
/// login non deve poterci navigare, nemmeno digitando l'URL a mano (rilevante
/// soprattutto sul web, dove l'URL è visibile e modificabile).
///
/// Questo provider dipende da `authControllerProvider` (`ref.watch`, non
/// `ref.read`): ogni volta che lo stato di autenticazione cambia (login,
/// logout, refresh fallito...) Riverpod ricrea l'intero `GoRouter`, che
/// applica subito la regola di redirect aggiornata alla posizione corrente.
/// Per un'app così piccola (poche schermate, nessuna navigazione profonda
/// da preservare) è il modo più semplice da seguire; un'app più grande
/// userebbe invece un `Listenable` dedicato per aggiornare un router già
/// esistente senza doverlo ricreare da zero.
final routerProvider = Provider<GoRouter>((ref) {
  final authState = ref.watch(authControllerProvider);

  return GoRouter(
    initialLocation: '/',
    redirect: (context, state) {
      final location = state.matchedLocation;
      final goingToSplash = location == '/';
      final goingToLogin = location == '/login';
      final goingToActivate = location == '/user-activation';
      final goingToForgotPassword = location == '/forgot-password';
      final goingToResetPassword = location == '/reset-password';

      switch (authState.status) {
        case AuthStatus.unknown:
          // Stiamo ancora controllando se c'è una sessione da riprendere
          // (vedi `AuthController._bootstrap`): si resta sullo splash finché
          // non sappiamo se mandare l'utente al login o al profilo, per
          // evitare un lampo della schermata di login a chi in realtà ha
          // già una sessione valida salvata.
          return goingToSplash ? null : '/';
        case AuthStatus.unauthenticated:
          return (goingToLogin ||
                  goingToActivate ||
                  goingToForgotPassword ||
                  goingToResetPassword)
              ? null
              : '/login';
        case AuthStatus.authenticated:
          // Solo la creazione utenti richiede il permesso a livello di
          // rotta: l'elenco ("/users") è una rubrica aperta a chiunque sia
          // autenticato, la modifica dei ruoli lì dentro si nasconde da
          // sola per chi non ha il permesso (vedi manage_users_screen.dart).
          if (location == '/users/new' &&
              !(authState.claims?.hasPermission('users:manage') ?? false)) {
            return '/home';
          }
          // Dopo il login si atterra sulla home (un punto di partenza da
          // cui raggiungere le varie aree dell'app), non più dritti sul
          // profilo — quello resta una delle destinazioni raggiungibili
          // da lì, non più la prima schermata.
          return (goingToLogin || goingToSplash) ? '/home' : null;
      }
    },
    routes: [
      GoRoute(
        path: '/home',
        pageBuilder: (context, state) =>
            _fadeTransitionPage(const HomeScreen()),
      ),
      GoRoute(
        path: '/users/new',
        pageBuilder: (context, state) =>
            _fadeTransitionPage(const CreateUserScreen()),
      ),
      GoRoute(
        path: '/users',
        pageBuilder: (context, state) =>
            _fadeTransitionPage(const ManageUsersScreen()),
      ),
      GoRoute(
        path: '/',
        pageBuilder: (context, state) =>
            _fadeTransitionPage(const SplashScreen()),
      ),
      GoRoute(
        path: '/login',
        pageBuilder: (context, state) =>
            _fadeTransitionPage(const LoginScreen()),
      ),
      GoRoute(
        path: '/user-activation',
        pageBuilder: (context, state) => _fadeTransitionPage(
          ActivateAccountScreen(
            inviteToken: state.uri.queryParameters['token'],
          ),
        ),
      ),
      GoRoute(
        path: '/forgot-password',
        pageBuilder: (context, state) =>
            _fadeTransitionPage(const ForgotPasswordScreen()),
      ),
      GoRoute(
        path: '/reset-password',
        pageBuilder: (context, state) => _fadeTransitionPage(
          ResetPasswordScreen(resetToken: state.uri.queryParameters['token']),
        ),
      ),
      GoRoute(
        path: '/profilo',
        pageBuilder: (context, state) =>
            _fadeTransitionPage(const ProfileScreen()),
      ),
    ],
  );
});
