// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for French (`fr`).
class AppLocalizationsFr extends AppLocalizations {
  AppLocalizationsFr([String locale = 'fr']) : super(locale);

  @override
  String get biking => 'Biking';

  @override
  String get canoeing => 'Canoeing';

  @override
  String get walking => 'Walking';

  @override
  String get about => 'Informations';

  @override
  String get appearance => 'Appearance';

  @override
  String get account => 'Account';

  @override
  String get account_delete_confirm =>
      'Vous êtes sur le point de supprimer votre compte. Tous vos itinéraires seront également supprimés. Voulez-vous continuer ?';

  @override
  String get account_privacy => 'Protection de la vie privée';

  @override
  String activity(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Activités',
      one: 'Activité',
    );
    return '$_temp0';
  }

  @override
  String get add_bio => 'Ajouter une description';

  @override
  String get add_entry => 'Ajouter une entrée';

  @override
  String get add_to_list => 'Ajouter à une liste';

  @override
  String get add_waypoint => 'Ajouter un point de passage';

  @override
  String get added_trail_to => 'Ajouter un itinéraire à';

  @override
  String get added_trails_to => 'Ajouter les itinéraires à';

  @override
  String get after => 'Après';

  @override
  String get all => 'All';

  @override
  String get all_activities => 'Toutes les activités';

  @override
  String get allow_auto_geolocate =>
      'Begin drawing a new trail from your current location';

  @override
  String get alphabetical => 'Alphabétique';

  @override
  String get already_account => 'Déjà un compte ?';

  @override
  String get altitude => 'Altitude';

  @override
  String get amenity => 'Amenity';

  @override
  String get api_documentation => 'Documentation API';

  @override
  String get api_tokens => 'API Tokens';

  @override
  String get api_tokens_hint =>
      'API Tokens can be used to grant 3rd party applications access to your wanderer account.';

  @override
  String get apply_user_settings => 'Apply user settings';

  @override
  String get attraction => 'Attraction';

  @override
  String get author => 'Auteur';

  @override
  String get avatar => 'Avatar';

  @override
  String get average_speed => 'Vitesse Moy.';

  @override
  String get avoid_bad_surfaces => 'Éviter les mauvaises surfaces';

  @override
  String get back => 'Retour';

  @override
  String get back_to_login => 'Retour à la page de connexion';

  @override
  String get bakery => 'Boulangerie';

  @override
  String get barrier => 'Barrière';

  @override
  String get basic_info => 'Informations de base';

  @override
  String get basque => 'Basque';

  @override
  String get before => 'Avant le';

  @override
  String get behavior => 'Behavior';

  @override
  String get bicycle_parking => 'Parking vélo';

  @override
  String get bicycle_rental => 'Location de vélos';

  @override
  String get bicycle_shop => 'Magasin de vélos';

  @override
  String get bike_type => 'Type de vélo';

  @override
  String get bus_stop => 'Arrêt de bus';

  @override
  String get by => 'par';

  @override
  String get campsite => 'Camping';

  @override
  String get can => 'peut';

  @override
  String get cancel => 'Annuler';

  @override
  String get car => 'Voiture';

  @override
  String get car_motorcycle => 'Voiture/Moto';

  @override
  String card(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Tuiles',
      one: 'Tuile',
    );
    return '$_temp0';
  }

  @override
  String get categories => 'Catégories';

  @override
  String get category => 'Catégorie';

  @override
  String get change => 'Changement';

  @override
  String get change_email => 'Changer d\'email';

  @override
  String get change_password => 'Changer de mot de passe';

  @override
  String get changelog => 'Journal des modifications';

  @override
  String get chinese => 'Chinois (simplifié)';

  @override
  String get clear_all => 'Cacher tous';

  @override
  String get climbing => 'Escalade';

  @override
  String get close => 'Fermer';

  @override
  String get collapse_trail_list => 'Collapse trail list';

  @override
  String comment(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Commentaires',
      one: 'Commentaire',
    );
    return '$_temp0';
  }

  @override
  String get completed => 'Terminé';

  @override
  String get completed_a_trail => 'a terminé un parcours';

  @override
  String get completed_tours => 'Tournées terminées';

  @override
  String get completion_status => 'Statut';

  @override
  String get confirm => 'Confirmer';

  @override
  String get confirm_deletion => 'Confirmer la suppression';

  @override
  String get confirm_publish => 'Confirmer la publication';

  @override
  String get confirm_share => 'Confirmer le partage';

  @override
  String get connect => 'Connecter';

  @override
  String get contribute => 'Contribuer';

  @override
  String get couldnt_start_navigation =>
      'Couldn\'t start navigation. Check your connection and try again.';

  @override
  String get location_services_disabled =>
      'Location services are disabled. Please enable GPS to use navigation.';

  @override
  String get location_permission_denied =>
      'Location permission is required for navigation.';

  @override
  String get location_permission_permanently_denied =>
      'Location permission is permanently denied. Please enable it in Settings.';

  @override
  String get copy_link => 'Copier le lien';

  @override
  String get create_new_list => 'Créer une nouvelle liste';

  @override
  String get create_waypoint => 'Créer un point de passage';

  @override
  String get creation_date => 'Date de création';

  @override
  String get crop => 'Recadrer';

  @override
  String get cross => 'Cross';

  @override
  String get current_password => 'Mot de passe actuel';

  @override
  String get cycling => 'Vélo';

  @override
  String get cycling_speed => 'Vitesse à vélo';

  @override
  String get czech => 'Czech';

  @override
  String get danger_zone => 'Zone de danger';

  @override
  String get date => 'Date';

  @override
  String get default_category => 'Catégorie par défaut';

  @override
  String get default_location => 'Lieu pas défaut';

  @override
  String get degrees => 'Degrés';

  @override
  String get delete => 'Supprimer';

  @override
  String get open => 'Open';

  @override
  String get delete_account => 'Supprimer le compte';

  @override
  String get delete_list_confirm =>
      'Voulez-vous vraiment supprimer cette liste ? Les itinéraires de la liste seront toujours disponibles.';

  @override
  String get delete_summit_log_confirm =>
      'Voulez-vous vraiment supprimer ce journal de sommet ? Cette action ne peut être annulée.';

  @override
  String get delete_trail_confirm =>
      'Voulez-vous vraiment supprimer cet itinéraire ? Cette action ne peut être annulée.';

  @override
  String get describe_your_trail => 'Décrivez votre itinéraire';

  @override
  String get description => 'Description';

  @override
  String get difficult => 'Difficile';

  @override
  String get difficulty => 'Difficulté';

  @override
  String get directions => 'Directions';

  @override
  String get display => 'Visualiser';

  @override
  String get display_as => 'Afficher en tant que';

  @override
  String get distance => 'Distance';

  @override
  String get documentation => 'Documentation';

  @override
  String get download => 'Télécharger';

  @override
  String get draw_a_route => 'Tracer un itinéraire';

  @override
  String get driving => 'Conduire';

  @override
  String get duplicate => 'Double';

  @override
  String get duration => 'Durée';

  @override
  String get dutch => 'Néerlandais';

  @override
  String get easy => 'Facile';

  @override
  String get edit => 'Editer';

  @override
  String get edit_entry => 'Éditer une entrée';

  @override
  String get edit_list => 'Éditer la liste';

  @override
  String get edit_route => 'Modifier l\'itinéraire';

  @override
  String get edit_waypoint => 'Éditer le point de passage';

  @override
  String get edited => 'modifié';

  @override
  String get elevation_gain => 'Dénivelé positif';

  @override
  String get elevation_loss => 'Dénivelé négatif';

  @override
  String get elevation_profile => 'Elevation Profile';

  @override
  String get email => 'Email';

  @override
  String get email_not_unique => 'This email address is already in use.';

  @override
  String get email_updated => 'Adresse mail mise à jour';

  @override
  String get email_verified => 'Email vérifié';

  @override
  String empty_activities(Object username) {
    return '$username n\'a encore aucune activité';
  }

  @override
  String empty_bio(Object username) {
    return '$username n\'a pas encore de description';
  }

  @override
  String get empty_feed => 'Votre fil est vide';

  @override
  String get empty_feed_explanation =>
      'Vos activités et celles des personnes que vous suivez apparaîtront ici';

  @override
  String empty_lists(Object username) {
    return '$username n\'a pas de liste publique';
  }

  @override
  String get enable_auto_routing => 'Activer le routage automatique';

  @override
  String get english => 'Anglais';

  @override
  String get entry => 'Entrée';

  @override
  String get error_copying_trail => 'Error copying trail';

  @override
  String get error_creating_user =>
      'Erreur durant la création de l\'utilisateur';

  @override
  String get error_deleting_token => 'Error deleting token';

  @override
  String get error_disabling_strava_integration =>
      'Erreur lors de la désactivation de l\'intégration Strava';

  @override
  String get error_during_login => 'Erreur durant la connexion';

  @override
  String get error_during_password_reset =>
      'Impossible d\'envoyer l\'e-mail de réinitialisation du mot de passe';

  @override
  String get error_exporting_trail =>
      'Erreur lors de l\'export de l\'itinéraire';

  @override
  String get error_generating_token => 'Error generating token';

  @override
  String get error_liking_trail => 'Erreur de like de l\'itinéraire';

  @override
  String get error_logging_in_to_hammerhead => 'Error logging in to Hammerhead';

  @override
  String get error_logging_in_to_komoot => 'Erreur de connexion à Komoot';

  @override
  String get error_posting_comment =>
      'Erreur lors de la publication du commentaire';

  @override
  String get error_printing_map => 'Erreur d\'impression de la carte';

  @override
  String get error_reading_file => 'Erreur de lecture du fichier';

  @override
  String get error_saving_list =>
      'Erreur lors de l\'enregistrement de la liste';

  @override
  String get error_saving_settings => 'Error saving settings';

  @override
  String get error_saving_trail =>
      'Erreur lors de l\'enregistrement de l\'itinéraire';

  @override
  String error_setting_up_integration(Object provider) {
    return 'Error setting up strava integration';
  }

  @override
  String get error_updating_hammerhead_integration =>
      'Error updating Hammerhead integration';

  @override
  String get error_updating_komoot_integration =>
      'Error updating komoot integration';

  @override
  String get error_updating_password =>
      'Erreur lors de la mise à jour du mot de passe';

  @override
  String get error_updating_strava_integration =>
      'Erreur lors de la mise à jour de l\'intégration Komoot';

  @override
  String get error_uploading_trail_to_hammerhead =>
      'Error uploading trail to Hammerhead';

  @override
  String get est_duration => 'Temps estimé';

  @override
  String get everyone_with_the_link => 'Tout le monde avec ce lien';

  @override
  String get expand_trail_list => 'Expand trail list';

  @override
  String get expiration => 'Expiration';

  @override
  String get expires => 'Expires';

  @override
  String get explore => 'Explorer';

  @override
  String get explore_some_trails => 'Explorer les itinéraires';

  @override
  String get exit_navigation => 'Exit';

  @override
  String get stop_navigation_confirm =>
      'Stop navigation and return to the trail?';

  @override
  String get search_this_area => 'Search this area';

  @override
  String get export => 'Exporter';

  @override
  String get export_all_trails => 'Exporter tous les itinéraires';

  @override
  String get favourite_sport => 'Activité favorite';

  @override
  String get features => 'Fonctionnalités';

  @override
  String get ferry => 'Ferry';

  @override
  String get file_format => 'Format du fichier';

  @override
  String file_too_big(Object file, Object size) {
    return 'Le fichier $file est trop volumineux (max. $size)';
  }

  @override
  String get filter_categories => 'Filtrer par catégories';

  @override
  String get filter_difficulty => 'Filtrer par difficulté';

  @override
  String get filter_tags => 'Filtrer par étiquettes';

  @override
  String get filter_trails => 'Filter trails';

  @override
  String get finish => 'Arrivée';

  @override
  String get fixed_speed => 'Vitesse fixe';

  @override
  String get focus_map_on => 'Centrer la carte sur';

  @override
  String get follow => 'Suivre';

  @override
  String get follow_request_pending => 'Demande en attente';

  @override
  String get followers => 'Abonné(e)s';

  @override
  String get following => 'Abonnements';

  @override
  String get food => 'Nourriture';

  @override
  String get food_drinks => 'Nourriture et Boissons';

  @override
  String get forgot_your_password => 'Mot de passe oublié ?';

  @override
  String get french => 'Français';

  @override
  String get from_file => 'Depuis un fichier';

  @override
  String get from_photos => 'Depuis les photos';

  @override
  String get from_url => 'Depuis une URL';

  @override
  String get garage => 'Garage';

  @override
  String get gas_station => 'Station-service';

  @override
  String get generate_new_token => 'Generate new token';

  @override
  String get german => 'Allemand';

  @override
  String get get_position_from_exif =>
      'Obtenir les coordonnées à partir des données EXIF';

  @override
  String get get_started => 'C\'est parti';

  @override
  String get grid => 'Grille';

  @override
  String get grocery_store => 'Épicerie';

  @override
  String get hammerhead_integration_after_date_hint =>
      'If your hammerhead account is already synced with other trail databases, such as komoot or Strava, start syncing your Hammerhead data may result in duplicates. To avoid this, you can set an start date below, meaning only activities recorded after this date will be synced.';

  @override
  String get heading => 'Titre';

  @override
  String get height => 'Hauteur';

  @override
  String get help => 'Aide';

  @override
  String get hero_section_0_text =>
      'Explore exciting trails, save your favorites, and experience the beauty of nature. Find your next adventure!';

  @override
  String get hero_section_1_heading => 'It seems there are no trails here yet.';

  @override
  String get hero_section_1_text =>
      'Here are some trails you might like. Or you can just go to the full list right now.';

  @override
  String get hero_section_1_text_alternative =>
      'Get started by saving your latest adventure.';

  @override
  String get hero_section_2_text =>
      'Did you know? You cannot only save you hiking trails. There are many categories for all your adventures.';

  @override
  String get hiking => 'Randonnée';

  @override
  String get home => 'Accueil';

  @override
  String get hotel => 'Hôtel';

  @override
  String get hungarian => 'Hongrois';

  @override
  String get hut => 'Cabane';

  @override
  String get hybrid => 'Hybride';

  @override
  String get icon => 'Icône';

  @override
  String get ignore_trails_before_date => 'Ignore trails before this date';

  @override
  String get imperial => 'Impérial';

  @override
  String get import => 'Importer';

  @override
  String get import_hint =>
      'Sélectionnez ou glissez des fichiers GPX, FIT, KML ou TCX ici...';

  @override
  String get include_description => 'Inclure la description';

  @override
  String get include_waypoints => 'Inclure les points de passage';

  @override
  String get integration_description_hammerhead =>
      'Syncs your Hammerhead tours with wanderer in regular intervals.';

  @override
  String get integration_description_komoot =>
      'Synchronisez vos Tours Komoot avec wanderer à intervalles réguliers.';

  @override
  String get integration_description_strava =>
      'Synchronisez vos itinéraires et vos activités Strava avec wanderer à intervalles réguliers.';

  @override
  String get integration_disabled => 'Intégration désactivée';

  @override
  String get integration_enabled => 'Intégration activée';

  @override
  String get integration_privacy_hint_original =>
      'Imported trails will maintain the same visibility they have on the external platform. For example, if the original trail was public, it will be public in wanderer, even if trails are private by default according to your privacy settings.';

  @override
  String get integration_privacy_hint_user =>
      'The original trail\'s visibility is discarded. Instead, the local privacy settings for trails are applied to all imported trails.';

  @override
  String get integrations => 'Intégrations';

  @override
  String get invalid_date => 'Date invalide';

  @override
  String get invalid_username => 'Nom d\'utilisateur invalide';

  @override
  String get italian => 'Italien';

  @override
  String get joined => 'Rejoint';

  @override
  String get keep_original => 'Keep original';

  @override
  String get keep_private => 'Keep private';

  @override
  String get language => 'Langue';

  @override
  String get last_used => 'Last used';

  @override
  String get latitude => 'Latitude';

  @override
  String layer(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Calques',
      one: 'Calque',
    );
    return '$_temp0';
  }

  @override
  String get license => 'Licence';

  @override
  String get like_status => 'Statut J\'aime';

  @override
  String get liked => 'Aimé';

  @override
  String get likes => '\"J\'aime\"';

  @override
  String get limited => 'Limité';

  @override
  String get link_copied => 'Lien copié !';

  @override
  String get linked_lists => 'Linked lists';

  @override
  String list(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Listes',
      one: 'Liste',
    );
    return '$_temp0';
  }

  @override
  String get list_not_shared => 'Non partagé avec quiconque';

  @override
  String get list_public_warning =>
      'Tous les itinéraires de cette liste seront publics.';

  @override
  String get list_saved_successfully => 'Liste enregistrée avec succès';

  @override
  String get list_share_warning =>
      'Le partage d\'une liste partage automatiquement toutes les pistes qu\'elle contient.';

  @override
  String get list_share_warning_update =>
      'Les itinéraires ajoutés seront partagés avec toutes les personnes qui ont accès à cette liste.';

  @override
  String get location => 'Localisation';

  @override
  String get locations => 'Locations';

  @override
  String get login => 'Connexion';

  @override
  String get login_details => 'Données de connexion';

  @override
  String get logout => 'Déconnexion';

  @override
  String get longitude => 'Longitude';

  @override
  String get loop => 'Boucle';

  @override
  String get make_one => 'Faites-en un !';

  @override
  String get make_thumbnail => 'Créer une miniature';

  @override
  String get map => 'Carte';

  @override
  String get map_style => 'Style de carte';

  @override
  String get max_hiking_difficulty => 'Difficulté maximale de marche';

  @override
  String get metric => 'Métrique';

  @override
  String get moderate => 'Moyenne';

  @override
  String get more => 'Plus';

  @override
  String get more_route_settings => 'Plus de paramètres pour l\'itinéraire';

  @override
  String get mountain => 'Montagne';

  @override
  String get mountain_pass => 'Col de montagne';

  @override
  String must_be_at_least_n_characters_long(Object n) {
    return 'Doit être composé d\'au moins $n caractères';
  }

  @override
  String must_be_at_most_n_characters_long(Object n) {
    return 'Doit être au maximum de $n caractères';
  }

  @override
  String get my_account => 'Mon profil';

  @override
  String get my_trails => 'Mes parcours';

  @override
  String in_distance(String distance) {
    return 'in $distance';
  }

  @override
  String n_days_ago(Object n) {
    return 'il y a $n jours';
  }

  @override
  String n_hours_ago(Object n) {
    return 'il y a $n heures';
  }

  @override
  String n_minutes_ago(Object n) {
    return 'il y a $n minutes';
  }

  @override
  String n_months_ago(Object n) {
    return 'il y a $n mois';
  }

  @override
  String n_seconds_ago(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'il y a $n secondes',
      zero: 'à l\'instant',
    );
    return '$_temp0';
  }

  @override
  String n_years_ago(Object n) {
    return 'il y a $n ans';
  }

  @override
  String get name => 'Nom de l\'itinéraire';

  @override
  String get navigate => 'Navigate';

  @override
  String get near => 'À proximité de';

  @override
  String get never => 'Never';

  @override
  String get new_list => 'Nouvelle liste';

  @override
  String get new_password => 'Nouveau mot de passe';

  @override
  String get new_password_error =>
      'Erreur lors de l\'enregistrement du nouveau mot de passe';

  @override
  String get new_password_success => 'Le nouveau mot de passe a été enregistré';

  @override
  String get new_password_text => 'Définir un nouveau mot de passe';

  @override
  String get new_token_generated => 'New API Token generated';

  @override
  String get new_trail => 'Nouvel itinéraire';

  @override
  String get no_account => 'Pas encore de compte ?';

  @override
  String get no_api_tokens => 'You have no API Tokens';

  @override
  String get no_comments_so_far => 'Aucun commentaire pour l\'instant';

  @override
  String get no_data => 'Pas de données';

  @override
  String get no_description_for_now => 'Pas de description pour le moment';

  @override
  String get no_gps_data_in_image => 'Aucune donnée GPS dans l\'image';

  @override
  String get no_grid => 'Aucune grille';

  @override
  String get no_notifications => 'Pas de notifications';

  @override
  String get no_photos_here => 'Aucune photo ici';

  @override
  String get no_preference => 'Tout afficher';

  @override
  String get no_results => 'Aucun résultat';

  @override
  String get no_routes_added => 'Aucun itinéraire ajouté';

  @override
  String get no_waypoints_yet => 'Pas encore de point de passage';

  @override
  String get norwegian => 'Norwegian';

  @override
  String get not_a_valid_email_address => 'Adresse email invalide';

  @override
  String get not_a_valid_url => 'URL non valide';

  @override
  String get not_completed => 'Non terminés';

  @override
  String notification_comment_mention(Object user) {
    return '$user vous a mentionné dans un commentaire';
  }

  @override
  String notification_list_create(Object user) {
    return '$user a créé une nouvelle liste';
  }

  @override
  String notification_list_share(Object user) {
    return '$user à partagé une liste avec vous';
  }

  @override
  String get notification_new_follower => 'Vous avez un nouvel abonné';

  @override
  String notification_summit_log_create(Object trail, Object user) {
    return '$user a créé un journal de sommet sur votre sentier \"$trail\"';
  }

  @override
  String notification_summit_log_mention(Object user) {
    return '$user vous a mentionné dans un journal de sommet';
  }

  @override
  String notification_trail_comment(Object trail, Object user) {
    return '$user a laissé un commentaire sur votre itinéraire $trail';
  }

  @override
  String notification_trail_create(Object user) {
    return '$user a créé un nouvel itinéraire';
  }

  @override
  String notification_trail_like(Object trail, Object user) {
    return '$user a aimé votre chemin \"$trail\"';
  }

  @override
  String notification_trail_mention(Object user) {
    return '$user vous a mentionné dans un chemin';
  }

  @override
  String notification_trail_share(Object user) {
    return '$user a partagé un itinéraire avec vous';
  }

  @override
  String get notifications => 'Notifications';

  @override
  String object_share_error(Object object) {
    return 'Un $object doit être public pour être partagé entre les instances.';
  }

  @override
  String get off => 'Désactivé';

  @override
  String get only_me => 'Seulement moi';

  @override
  String get open_in_new_tab => 'Ouvrir dans un nouvel onglet';

  @override
  String get or => 'ou';

  @override
  String get orientation => 'Orientation';

  @override
  String get paper_size => 'Taille de papier';

  @override
  String get paragraph => 'Paragraphe';

  @override
  String get parking => 'Stationnement';

  @override
  String get pause => 'Pause';

  @override
  String get password => 'Mot de passe';

  @override
  String get password_confirm => 'Confirmer le mot de passe';

  @override
  String get password_reset_sent =>
      'Un e-mail de réinitialisation de votre mot de passe a été envoyé';

  @override
  String get password_reset_text =>
      'Un lien de réinitialisation vous sera envoyé sur votre adresse e-mail.';

  @override
  String get password_updated => 'Mot de passe mis à jour';

  @override
  String get passwords_must_match =>
      'Les mots de passes doivent être identiques';

  @override
  String get photos => 'Photos';

  @override
  String get pick_a_trail => 'Choisir un itinéraire';

  @override
  String get planned_a_trail => 'a plannifié un itinéraire';

  @override
  String get planned_tours => 'Tournées planifiées';

  @override
  String get pois => 'POIs';

  @override
  String get polish => 'polonais';

  @override
  String get portuguese => 'Portugais';

  @override
  String get print => 'Imprimer';

  @override
  String get privacy => 'Confidentialité';

  @override
  String get private => 'Privé';

  @override
  String get profile => 'Profile';

  @override
  String get public => 'Publique';

  @override
  String get public_access => 'Accès public';

  @override
  String get public_share_everyone =>
      'Avec ce lien, n\'importe qui sur internet peut voir cet itinéraire';

  @override
  String get public_share_limited =>
      'Seules les personnes ayant un accès peuvent ouvrir ce lien';

  @override
  String get public_transport => 'Transport en commun';

  @override
  String get radius => 'Rayon';

  @override
  String get railway_station => 'Gare ferroviaire';

  @override
  String get reached_end_of_trail => 'You\'ve reached the end of the trail.';

  @override
  String get read_more => 'Voir plus';

  @override
  String get ready_to_join => 'Prêt à rejoindre';

  @override
  String get recalculate_elevation_data => 'Recalculer les données d\'altitude';

  @override
  String get recalculating_elevation_data_hint =>
      'Recalculer les données d\'altitude effacera les données d\'altitude existantes, si c\'est le cas, et les remplacera par les données de Valhalla.';

  @override
  String get register => 'Créer un compte';

  @override
  String get remote_users_cannot_edit =>
      'Les utilisateurs distants ne peuvent pas modifier';

  @override
  String get removed_trail_from => 'Enlever l\'itinéraire de';

  @override
  String get removed_trails_from => 'Enlever les itinéraires de';

  @override
  String get required => 'Requis';

  @override
  String get reset => 'Réinitialiser';

  @override
  String get reset_password => 'Réinitialiser le mot de passe';

  @override
  String get resume => 'Resume';

  @override
  String get reverse_direction => 'Inverser la direction';

  @override
  String get road => 'Route';

  @override
  String route(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Itinéraires',
      one: 'Itinéraire',
    );
    return '$_temp0';
  }

  @override
  String get route_point => 'Étape';

  @override
  String get russian => 'Russe';

  @override
  String get save => 'Sauvegarder';

  @override
  String get save_list => 'Sauvegarder la liste';

  @override
  String get save_trail => 'Sauvegarder l\'itinéraire';

  @override
  String get save_your_trail_first => 'Enregistrez d\'abord votre itinéraire';

  @override
  String get search => 'Search';

  @override
  String get search_cities => 'Recherche une ville';

  @override
  String get search_for_trails_places => 'Chercher un itinéraire ou un lieu';

  @override
  String get search_list => 'Search list';

  @override
  String get search_places => 'Chercher des lieux';

  @override
  String get search_trails => 'Chercher un itinéraire';

  @override
  String get select_date => 'Select date';

  @override
  String get select_list => 'Liste de choix';

  @override
  String get selected => 'sélectionné(s)';

  @override
  String get send_to => 'Send to...';

  @override
  String get set_private => 'Set private';

  @override
  String get set_public => 'Set public';

  @override
  String get settings => 'Paramètres';

  @override
  String get settings_notification_comment_mention =>
      'Quelqu’un vous a mentionné dans un commentaire';

  @override
  String get settings_notification_list_share =>
      'Quelqu\'un à partagé une liste avec vous';

  @override
  String get settings_notification_new_follower => 'Vous avez un nouvel abonné';

  @override
  String get settings_notification_summit_log_create =>
      'Quelqu\'un a créé un journal de sommet sur votre chemin';

  @override
  String get settings_notification_summit_log_mention =>
      'Quelqu’un vous a mentionné dans un journal de sommet';

  @override
  String get settings_notification_trail_comment =>
      'Quelqu\'un a commenté votre itinéraire';

  @override
  String get settings_notification_trail_like => 'Somone liked your trail';

  @override
  String get settings_notification_trail_mention =>
      'Quelqu’un vous a mentionné dans un chemin';

  @override
  String get settings_notification_trail_share =>
      'Quelqu\'un a partagé un itinéraire avec vous';

  @override
  String get settings_privacy_account_private =>
      'Vous seul pouvez voir votre profil. Vous n\'apparaîtrez pas dans les résultats de recherche. Les autres utilisateurs ne pourront pas vous suivre ni partager d\'itinéraire avec vous. Vous pourrez tout de même publier des itinéraires ou des listes.';

  @override
  String get settings_privacy_account_public =>
      'Tout le monde peut voir votre profil. Vous apparaîtrez dans les résultats de recherche. Les autres utilisateurs pourront vous suivre et partager des itinéraires avec vous.';

  @override
  String get settings_privacy_lists_private =>
      'Vos listes sont privées par défaut. Personne d\'autre que vous ne peut les voir. Vous pouvez modifier ce paramètre à tout moment pour les listes individuelles.';

  @override
  String get settings_privacy_lists_public =>
      'Vos listes sont publiques par défaut. Tout le monde peut les voir. Vous pouvez modifier ce paramètre à tout moment pour les listes individuelles.';

  @override
  String get settings_privacy_trails_private =>
      'Vos itinéraires sont privés par défaut. Personne d\'autre que vous ne peut les voir. Vous pouvez modifier ce paramètre à tout moment pour les itinéraires individuels.';

  @override
  String get settings_privacy_trails_public =>
      'Vos itinéraires sont publics par défaut. Tout le monde peut les voir. Vous pouvez modifier ce paramètre à tout moment pour les itinéraires individuels.';

  @override
  String get settings_saved => 'Configuration sauvegardée';

  @override
  String get share => 'Partager';

  @override
  String get share_profile => 'Partager le profil';

  @override
  String get share_this_list => 'Partager cette liste';

  @override
  String get share_this_trail => 'Partager l\'itinéraire';

  @override
  String get shared => 'Partagé';

  @override
  String get shared_by => 'Partagé par';

  @override
  String get shared_with => 'Partagé avec';

  @override
  String get shelter => 'Refuge';

  @override
  String get shortest => 'la plus courte';

  @override
  String get show_in_overview => 'Voir dans l\'aperçu';

  @override
  String get show_less => 'Afficher moins';

  @override
  String get show_on_map => 'Voir sur la carte';

  @override
  String get shower => 'Douche';

  @override
  String get skiing => 'Ski';

  @override
  String get slogan => 'Sauvegarder vos aventures !';

  @override
  String get slope => 'Pente';

  @override
  String get someone => 'Quelqu\'un';

  @override
  String get sort => 'Trier';

  @override
  String get spanish => 'Espagnol';

  @override
  String get speed => 'Vitesse';

  @override
  String get start => 'Départ';

  @override
  String get statistics => 'Statistiques';

  @override
  String get stop_drawing => 'Arrêter de tracer';

  @override
  String get stop_editing => 'Arrêter la modification';

  @override
  String get strava_integration_after_date_hint =>
      'Si votre compte a une grande quantité d\'activités, vous pouvez rencontrer la limite d\'utilisation de l\'API de Strava vous empêchant de synchroniser toutes les activités en même temps. Pour atténuer ce problème, vous pouvez définir une date \"Après-\" ci-dessous afin que seules les activités qui ont été enregistrées après cette date soient synchronisées.';

  @override
  String get subcategories => 'Sous-catégories';

  @override
  String get subway_stop => 'Bouche de métro';

  @override
  String get summit => 'Sommet';

  @override
  String get summit_book => 'Liste des ascensions';

  @override
  String get table => 'Tableau';

  @override
  String get tags => 'Étiquettes';

  @override
  String get text => 'Texte';

  @override
  String get theme_dark => 'Dark';

  @override
  String get theme_light => 'Light';

  @override
  String get theme_system => 'Follow system';

  @override
  String get tilesets => 'Tuiles personnalisés';

  @override
  String get time => 'Time';

  @override
  String get toilets => 'Toilettes';

  @override
  String get top_speed => 'Vitesse maximale';

  @override
  String get tourism => 'Tourisme';

  @override
  String trail(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Itinéraires',
      one: 'Itinéraire',
    );
    return '$_temp0';
  }

  @override
  String get trail_copied_successfully => 'trail copied successfully';

  @override
  String get trail_has_no_gpx => 'This trail has no GPX data.';

  @override
  String get trail_not_in_list => 'Trail is not in any list';

  @override
  String get trail_not_shared => 'L\'itinéraire n\'a pas été partagé';

  @override
  String get trail_saved_successfully => 'Itinéraire enregistrée';

  @override
  String get trails_for_you => 'Itinéraire pour vous';

  @override
  String get tram_stop => 'Arrêt de tram';

  @override
  String get unchanged => 'pas de modification';

  @override
  String get units => 'Unités';

  @override
  String get unlink => 'Délier';

  @override
  String get upload_file => 'Importer un fichier';

  @override
  String get upload_gpx => 'Envoyer un GPX';

  @override
  String get upload_new_file => 'Importer un fichier';

  @override
  String get uploaded => 'importé';

  @override
  String get uploaded_trail_to_hammerhead =>
      'Successfully uploaded trail to Hammerhead';

  @override
  String get use_hills => 'Utiliser les collines';

  @override
  String get use_roads => 'Utiliser les routes';

  @override
  String get users => 'Users';

  @override
  String get username => 'Nom d\'utilisateur';

  @override
  String get username_not_unique =>
      'This username is already taken. Please try another.';

  @override
  String get view => 'Afficher';

  @override
  String get viewpoint => 'Point de vue';

  @override
  String get visibilty => 'Visibilité';

  @override
  String get visibilty_status => 'État de visibilité';

  @override
  String get walking_speed => 'Vitesse de marche';

  @override
  String get water => 'Eau';

  @override
  String get web => 'Web';

  @override
  String waypoints(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Points de passage',
      one: 'Point de passage',
    );
    return '$_temp0';
  }

  @override
  String get welcome_to => 'Welcome to';

  @override
  String get width => 'Largeur';

  @override
  String get wrong_username_or_password =>
      'Nom d\'utilisateur ou mot de passe incorrect';

  @override
  String get you_can => 'Vous pouvez';

  @override
  String get you_have_arrived => 'You\'ve arrived';

  @override
  String get settings_categories_confirm_disable_title => 'Hide this category?';

  @override
  String get settings_categories_confirm_disable_subcategory_title =>
      'Hide this subcategory?';

  @override
  String settings_categories_confirm_disable_body(int count) {
    return '$count of your trails use this category. They will stay published but this filter will be hidden.';
  }

  @override
  String get settings_categories_confirm_view_trails => 'View trails';

  @override
  String get settings_categories_confirm_disable_confirm => 'Disable anyway';

  @override
  String get settings_categories_empty_title => 'No subcategories';

  @override
  String get settings_categories_empty_body =>
      'This category has no subcategories to configure.';
}
