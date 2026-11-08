import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/actor.dart';
import 'package:wanderer/models/user.dart';

void main() {
  final created = DateTime.utc(2025, 11, 17);

  User userWithActorIri(String iri) => User(
    id: 'u1',
    collectionId: '_pb_users_auth_',
    collectionName: 'users',
    created: created,
    updated: created,
    username: 'bob',
    email: 'bob@example.com',
    emailVisibility: false,
    verified: true,
    expand: UserExpand(
      actor: Actor(
        id: 'a1',
        collectionId: 'actors',
        collectionName: 'activitypub_actors',
        created: created,
        updated: created,
        username: 'bob',
        preferredUsername: 'bob',
        iri: iri,
        inbox: '$iri/inbox',
        publicKey: '',
        lastFetched: '',
        user: 'u1',
      ),
    ),
  );

  // #1411: the actor IRI keeps the address the account was created under.
  // After the instance moved, restoring the session against it found no
  // cookie on every cold start.
  test('keeps the URL the user signed in through, not the actor IRI', () {
    final entity = userWithActorIri(
      'http://192.168.1.20:3000/api/v1/activitypub/user/bob',
    ).toEntity(serverUrl: 'https://wanderer.example.com');

    expect(entity.serverUrl, 'https://wanderer.example.com');
    expect(entity.iri, 'http://192.168.1.20:3000/api/v1/activitypub/user/bob');
  });
}
