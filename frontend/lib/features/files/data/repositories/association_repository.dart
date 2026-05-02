import '../../../../core/api_client.dart';
import '../models/association_model.dart';

class AssociationRepository {
  final ApiClient _client;

  AssociationRepository(this._client);

  Future<List<AssociationModel>> listAll() async {
    final res = await _client.get<Map<String, dynamic>>('/associations');
    final data = res.data!['data'] as Map<String, dynamic>;
    final items = data['associations'] as List<dynamic>;
    return items
        .map((e) => AssociationModel.fromJson(e as Map<String, dynamic>))
        .toList();
  }
}
