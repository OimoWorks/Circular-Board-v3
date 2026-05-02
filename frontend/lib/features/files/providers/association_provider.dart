import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../auth/presentation/auth_provider.dart';
import '../data/models/association_model.dart';
import '../data/repositories/association_repository.dart';

final associationRepositoryProvider = Provider<AssociationRepository>((ref) {
  final client = ref.watch(apiClientProvider);
  return AssociationRepository(client);
});

final allAssociationsProvider = FutureProvider<List<AssociationModel>>((ref) async {
  final repo = ref.watch(associationRepositoryProvider);
  return repo.listAll();
});
