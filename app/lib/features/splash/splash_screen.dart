import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';

/// Mostrata solo per l'istante in cui l'app non sa ancora se c'è una
/// sessione da riprendere (`AuthStatus.unknown` — vedi `router.dart` e
/// `AuthController._bootstrap`). Dura tipicamente pochi decimi di secondo:
/// il tempo di un'unica chiamata a `POST /v1/refresh` se c'è un refresh
/// token salvato, altrimenti è quasi istantanea.
class SplashScreen extends StatelessWidget {
  const SplashScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const CircularProgressIndicator(),
            const SizedBox(height: 16),
            Text('app.title'.tr(), style: Theme.of(context).textTheme.titleMedium),
          ],
        ),
      ),
    );
  }
}
