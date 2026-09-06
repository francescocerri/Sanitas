/// Rispecchia la risposta di `GET /v1/me` (vedi `docs/adr/0022-frontend-flutter.md`
/// per il contratto completo dell'API di `registry`).
class UserProfile {
  const UserProfile({
    required this.email,
    required this.username,
    required this.roles,
    required this.createdAt,
  });

  factory UserProfile.fromJson(Map<String, dynamic> json) {
    return UserProfile(
      email: json['email'] as String,
      username: json['username'] as String,
      roles: (json['roles'] as List<dynamic>).cast<String>(),
      createdAt: DateTime.parse(json['created_at'] as String),
    );
  }

  final String email;
  final String username;
  final List<String> roles;
  final DateTime createdAt;
}
