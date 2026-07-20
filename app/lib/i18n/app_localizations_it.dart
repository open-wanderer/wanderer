// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Italian (`it`).
class AppLocalizationsIt extends AppLocalizations {
  AppLocalizationsIt([String locale = 'it']) : super(locale);

  @override
  String get biking => 'Biking';

  @override
  String get canoeing => 'Canoeing';

  @override
  String get walking => 'Walking';

  @override
  String get about => 'Su di noi';

  @override
  String get appearance => 'Appearance';

  @override
  String get account => 'Account';

  @override
  String get account_delete_confirm =>
      'Stai per eliminare il tuo account. Tutti i tuoi percorsi saranno cancellati. Vuoi procedere?';

  @override
  String get account_privacy => 'Privacy dell\'account';

  @override
  String activity(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Attivitá',
      one: 'Attività',
    );
    return '$_temp0';
  }

  @override
  String get add_bio => 'Aggiungere Biografia';

  @override
  String get add_entry => 'Aggiungi voce';

  @override
  String get add_to_list => 'Aggiungi alla lista';

  @override
  String get add_waypoint => 'Aggiungi un punto di passaggio';

  @override
  String get added_trail_to => 'Percorso aggiunto a';

  @override
  String get added_trails_to => 'Percorsi aggiunto a';

  @override
  String get after => 'Dopo';

  @override
  String get all => 'All';

  @override
  String get all_activities => 'Tutte le Attività';

  @override
  String get allow_auto_geolocate =>
      'Begin drawing a new trail from your current location';

  @override
  String get alphabetical => 'Alfabetico';

  @override
  String get already_account => 'Hai già un account?';

  @override
  String get altitude => 'Altitudine';

  @override
  String get amenity => 'Amenity';

  @override
  String get api_documentation => 'Documentazione API';

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
  String get author => 'Autore';

  @override
  String get avatar => 'Avatar';

  @override
  String get average_speed => 'Vel. Media';

  @override
  String get avoid_bad_surfaces => 'Avoid Bad Surfaces';

  @override
  String get back => 'Indietro';

  @override
  String get back_to_login => 'Indietro al Login';

  @override
  String get bakery => 'Bakery';

  @override
  String get barrier => 'Barrier';

  @override
  String get basic_info => 'Informazioni di base';

  @override
  String get basque => 'Basque';

  @override
  String get before => 'Prima';

  @override
  String get behavior => 'Behavior';

  @override
  String get bicycle_parking => 'Bicycle Parking';

  @override
  String get bicycle_rental => 'Bicycle Rental';

  @override
  String get bicycle_shop => 'Bicycle Shop';

  @override
  String get bike_type => 'Bike Type';

  @override
  String get bus_stop => 'Bus stop';

  @override
  String get by => 'di';

  @override
  String get campsite => 'Campsite';

  @override
  String get can => 'può';

  @override
  String get cancel => 'Annulla';

  @override
  String get car => 'Car';

  @override
  String get car_motorcycle => 'Car/Motorcycle';

  @override
  String card(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Carte',
      one: 'Carta',
    );
    return '$_temp0';
  }

  @override
  String get categories => 'Categorie';

  @override
  String get category => 'Categoria';

  @override
  String get change => 'Modifica';

  @override
  String get change_email => 'Cambia email';

  @override
  String get change_password => 'Cambia password';

  @override
  String get changelog => 'Changelog';

  @override
  String get chinese => 'Cinese (semplificato)';

  @override
  String get clear_all => 'Clear all';

  @override
  String get climbing => 'Climbing';

  @override
  String get close => 'Chiudi';

  @override
  String get collapse_trail_list => 'Collapse trail list';

  @override
  String comment(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Commenti',
      one: 'Commento',
    );
    return '$_temp0';
  }

  @override
  String get completed => 'Completato';

  @override
  String get completed_a_trail => 'percorso completato';

  @override
  String get completed_tours => 'Completed tours';

  @override
  String get completion_status => 'Stato di completamento';

  @override
  String get confirm => 'Confermare';

  @override
  String get confirm_deletion => 'Conferma eliminazione';

  @override
  String get confirm_publish => 'Conferma pubblicazione';

  @override
  String get confirm_share => 'Conferma condivisione';

  @override
  String get connect => 'Connect';

  @override
  String get contribute => 'Contribuisci';

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
  String get location_unavailable =>
      'Unable to determine your location. Please try again.';

  @override
  String get copy_link => 'Copia link';

  @override
  String get create_new_list => 'Crea nuova lista';

  @override
  String get create_waypoint => 'Create waypoint';

  @override
  String get creation_date => 'Data di creazione';

  @override
  String get crop => 'Crop';

  @override
  String get cross => 'Cross';

  @override
  String get current_password => 'Password attuale';

  @override
  String get cycling => 'Ciclismo';

  @override
  String get cycling_speed => 'Cycling Speed';

  @override
  String get czech => 'Czech';

  @override
  String get danger_zone => 'Zona di pericolo';

  @override
  String get date => 'Data';

  @override
  String get default_category => 'Categoria predefinita';

  @override
  String get default_location => 'Posizione predefinita';

  @override
  String get degrees => 'Gradi';

  @override
  String get delete => 'Elimina';

  @override
  String get open => 'Open';

  @override
  String get delete_account => 'Elimina account';

  @override
  String get delete_list_confirm =>
      'Vuoi davvero cancellare questa lista? I percorsi presenti nella lista saranno ancora disponibili.';

  @override
  String get delete_summit_log_confirm =>
      'Do you really want to delete this summit log? This action cannot be undone.';

  @override
  String get delete_trail_confirm =>
      'Vuoi davvero cancellare questo percorso? Questa azione non può essere revocata.';

  @override
  String get describe_your_trail => 'Descrivi il tuo percorso';

  @override
  String get description => 'Descrizione';

  @override
  String get difficult => 'Difficile';

  @override
  String get difficulty => 'Difficoltà';

  @override
  String get directions => 'Indicazioni';

  @override
  String get display => 'Visualizza';

  @override
  String get display_as => 'Visualizza come';

  @override
  String get distance => 'Distanza';

  @override
  String get documentation => 'Documentazione';

  @override
  String get download => 'Scarica';

  @override
  String get draw_a_route => 'Disegna un percorso';

  @override
  String get driving => 'Guida';

  @override
  String get duplicate => 'Duplicate';

  @override
  String get duration => 'Durata';

  @override
  String get dutch => 'Olandese';

  @override
  String get easy => 'Facile';

  @override
  String get edit => 'Modifica';

  @override
  String get edit_entry => 'Modifica voce';

  @override
  String get edit_list => 'Modifica lista';

  @override
  String get edit_route => 'Edit route';

  @override
  String get edit_waypoint => 'Modifica waypoint';

  @override
  String get edited => 'modificato';

  @override
  String get elevation_gain => 'Guadagno di quota';

  @override
  String get elevation_loss => 'Perdita di quota';

  @override
  String get elevation_profile => 'Elevation Profile';

  @override
  String get email => 'Email';

  @override
  String get email_not_unique => 'This email address is already in use.';

  @override
  String get email_updated => 'Indirizzo email aggiornato';

  @override
  String get email_verified => 'Email verified';

  @override
  String empty_activities(Object username) {
    return '$username non ha ancora attività';
  }

  @override
  String empty_bio(Object username) {
    return '$username non ha ancora aggiunto una biografia';
  }

  @override
  String get empty_feed => 'Your feed is empty';

  @override
  String get empty_feed_explanation =>
      'Activities by you or people you follow will appear here';

  @override
  String empty_lists(Object username) {
    return '$username non ha liste pubbliche';
  }

  @override
  String get enable_auto_routing => 'Enable auto-routing';

  @override
  String get english => 'Inglese';

  @override
  String get entry => 'Voce';

  @override
  String get error_copying_trail => 'Error copying trail';

  @override
  String get error_creating_user => 'Errore nella creazione dell\'utente';

  @override
  String get error_deleting_token => 'Error deleting token';

  @override
  String get error_disabling_strava_integration =>
      'Error disabling strava integration';

  @override
  String get error_during_login => 'Errore durante il login';

  @override
  String get error_during_password_reset =>
      'Impossibile inviare email per ripristinare la password';

  @override
  String get error_exporting_trail =>
      'Errore durante l\'esportazione del percorso';

  @override
  String get error_generating_token => 'Error generating token';

  @override
  String get error_liking_trail => 'Error liking trail';

  @override
  String get error_logging_in_to_hammerhead => 'Error logging in to Hammerhead';

  @override
  String get error_logging_in_to_komoot => 'Error logging in to komoot';

  @override
  String get error_posting_comment => 'Errore pubblicando il commento';

  @override
  String get error_printing_map => 'Errore durante la stampa della mappa';

  @override
  String get error_reading_file => 'Errore durante la lettura del file';

  @override
  String get error_saving_list => 'Errore salvando la lista';

  @override
  String get error_saving_settings => 'Error saving settings';

  @override
  String get error_saving_trail => 'Errore nel salvataggio del percorso';

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
      'Errore nell\'aggiornamento della password';

  @override
  String get error_updating_strava_integration =>
      'Error updating komoot integration';

  @override
  String get error_uploading_trail_to_hammerhead =>
      'Error uploading trail to Hammerhead';

  @override
  String get est_duration => 'Durata stimata';

  @override
  String get everyone_with_the_link => 'Everyone with the link';

  @override
  String get expand_trail_list => 'Expand trail list';

  @override
  String get expiration => 'Expiration';

  @override
  String get expires => 'Expires';

  @override
  String get explore => 'Esplora';

  @override
  String get explore_some_trails => 'Esplora alcuni percorsi';

  @override
  String get exit_navigation => 'Exit';

  @override
  String get stop_navigation_confirm =>
      'Stop navigation and return to the trail?';

  @override
  String get stop_recording => 'Interrompi registrazione';

  @override
  String get stop_recording_confirm => 'Interrompere la registrazione?';

  @override
  String get search_this_area => 'Search this area';

  @override
  String get export => 'Esporta';

  @override
  String get export_all_trails => 'Esporta tutti i percorsi';

  @override
  String get favourite_sport => 'Attività preferita';

  @override
  String get features => 'Caratteristiche';

  @override
  String get ferry => 'Ferry';

  @override
  String get file_format => 'Formato del file';

  @override
  String file_too_big(Object file, Object size) {
    return 'File $file è troppo grande (max. $size)';
  }

  @override
  String get filter_categories => 'Filtrare categorie';

  @override
  String get filter_difficulty => 'Filtrare difficoltà';

  @override
  String get filter_tags => 'Filter tags';

  @override
  String get filter_trails => 'Filter trails';

  @override
  String get finish => 'Finish';

  @override
  String get finish_disabled_hint =>
      'Add at least 2 anchors to finish your route.';

  @override
  String get fixed_speed => 'Fixed Speed';

  @override
  String get focus_map_on => 'Focus sulla mappa';

  @override
  String get follow => 'Seguire';

  @override
  String get follow_request_pending => 'Richiesta in sospeso';

  @override
  String get followers => 'Seguaci';

  @override
  String get following => 'Seguendo';

  @override
  String get food => 'Food';

  @override
  String get food_drinks => 'Food & Drinks';

  @override
  String get forgot_your_password => 'Password dimenticata?';

  @override
  String get french => 'Francese';

  @override
  String get from_file => 'From file';

  @override
  String get from_photos => 'From Photos';

  @override
  String get from_url => 'From URL';

  @override
  String get garage => 'Garage';

  @override
  String get gas_station => 'Gas station';

  @override
  String get generate_new_token => 'Generate new token';

  @override
  String get german => 'Tedesco';

  @override
  String get get_position_from_exif => 'Ottieni posizione da dati EXIF';

  @override
  String get get_started => 'Get started';

  @override
  String get grid => 'Griglia';

  @override
  String get grocery_store => 'Grocery store';

  @override
  String get hammerhead_integration_after_date_hint =>
      'If your hammerhead account is already synced with other trail databases, such as komoot or Strava, start syncing your Hammerhead data may result in duplicates. To avoid this, you can set an start date below, meaning only activities recorded after this date will be synced.';

  @override
  String get heading => 'Heading';

  @override
  String get height => 'Height';

  @override
  String get help => 'Aiuto';

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
  String get hiking => 'Escursionismo';

  @override
  String get home => 'Home';

  @override
  String get hotel => 'Hotel';

  @override
  String get hungarian => 'Ungherese';

  @override
  String get hut => 'Hut';

  @override
  String get hybrid => 'Hybrid';

  @override
  String get icon => 'Icona';

  @override
  String get ignore_trails_before_date => 'Ignore trails before this date';

  @override
  String get imperial => 'Imperiale';

  @override
  String get import => 'Importa';

  @override
  String get import_hint =>
      'Seleziona o trascina qui i file GPX, FIT, KML o TCX...';

  @override
  String get include_description => 'Adotta descrizione';

  @override
  String get include_waypoints => 'Includi waypoint';

  @override
  String get integration_description_hammerhead =>
      'Syncs your Hammerhead tours with wanderer in regular intervals.';

  @override
  String get integration_description_komoot =>
      'Syncs your komoot tours with wanderer in regular intervals.';

  @override
  String get integration_description_strava =>
      'Syncs your strava routes & activities with wanderer in regular intervals.';

  @override
  String get integration_disabled => 'integration disabled';

  @override
  String get integration_enabled => 'integration enabled';

  @override
  String get integration_privacy_hint_original =>
      'Imported trails will maintain the same visibility they have on the external platform. For example, if the original trail was public, it will be public in wanderer, even if trails are private by default according to your privacy settings.';

  @override
  String get integration_privacy_hint_user =>
      'The original trail\'s visibility is discarded. Instead, the local privacy settings for trails are applied to all imported trails.';

  @override
  String get integrations => 'Integrations';

  @override
  String get invalid_date => 'Data non valida';

  @override
  String get invalid_username => 'Nome utente non valido';

  @override
  String get italian => 'Italiano';

  @override
  String get joined => 'Aggiunto';

  @override
  String get keep_original => 'Keep original';

  @override
  String get keep_private => 'Keep private';

  @override
  String get language => 'Lingua';

  @override
  String get last_used => 'Last used';

  @override
  String get latitude => 'Latitudine';

  @override
  String layer(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Layers',
      one: 'Layer',
    );
    return '$_temp0';
  }

  @override
  String get license => 'Licenza';

  @override
  String get like_status => 'Like Status';

  @override
  String get liked => 'Liked';

  @override
  String get likes => 'Likes';

  @override
  String get limited => 'Limited';

  @override
  String get link_copied => 'Link copiato';

  @override
  String get linked_lists => 'Linked lists';

  @override
  String list(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Liste',
      one: 'Lista',
    );
    return '$_temp0';
  }

  @override
  String get list_not_shared => 'Non condivisa';

  @override
  String get list_public_warning =>
      'Tutti i percorsi in questa lista diventeranno pubblici.';

  @override
  String get list_saved_successfully => 'Lista salvata con successo';

  @override
  String get list_share_warning =>
      'Condividere una lista automaticamente condivide tutti i percorsi che contiene.';

  @override
  String get list_share_warning_update =>
      'I percorsi aggiunti saranno condivisi con tutti gli utenti che hanno accesso alla lista.';

  @override
  String get location => 'Posizione';

  @override
  String get locations => 'Locations';

  @override
  String get center_on_my_location => 'Center on my location';

  @override
  String get location_tracking_notification_title => 'Wanderer';

  @override
  String get location_tracking_notification_text => 'Recording your trail';

  @override
  String get login => 'Login';

  @override
  String get login_details => 'Dettagli del login';

  @override
  String get logout => 'Logout';

  @override
  String get longitude => 'Longitudine';

  @override
  String get loop => 'Loop';

  @override
  String get make_one => 'Creane uno!';

  @override
  String get make_thumbnail => 'Imposta miniatura';

  @override
  String get map => 'Mappa';

  @override
  String get map_style => 'Map style';

  @override
  String get max_hiking_difficulty => 'Max. Hiking Difficulty';

  @override
  String get metric => 'Metrico';

  @override
  String get moderate => 'Moderato';

  @override
  String get more => 'More';

  @override
  String get more_route_settings => 'More route settings';

  @override
  String get mountain => 'Mountain';

  @override
  String get mountain_pass => 'Mountain pass';

  @override
  String must_be_at_least_n_characters_long(Object n) {
    return 'Deve essere lungo almeno $n caratteri';
  }

  @override
  String must_be_at_most_n_characters_long(Object n) {
    return 'Must be at most $n characters long';
  }

  @override
  String get my_account => 'Il mio account';

  @override
  String get my_trails => 'My trails';

  @override
  String in_distance(String distance) {
    return 'in $distance';
  }

  @override
  String n_days_ago(Object n) {
    return '$n giorni fa';
  }

  @override
  String n_hours_ago(Object n) {
    return '$n ore fa';
  }

  @override
  String n_minutes_ago(Object n) {
    return '$n minuti fa';
  }

  @override
  String n_months_ago(Object n) {
    return '$n mesi fa';
  }

  @override
  String n_seconds_ago(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '$n secondi fa',
      zero: 'solo ora',
    );
    return '$_temp0';
  }

  @override
  String n_years_ago(Object n) {
    return '$n anni fa';
  }

  @override
  String get name => 'Nome';

  @override
  String get navigate => 'Navigate';

  @override
  String get near => 'Vicino';

  @override
  String get never => 'Never';

  @override
  String get new_list => 'Nuova lista';

  @override
  String get new_password => 'Nuova password';

  @override
  String get new_password_error => 'Errore configurando la nuova password';

  @override
  String get new_password_success => 'La nuova password è stata configurata';

  @override
  String get new_password_text => 'Definire una nuova password';

  @override
  String get new_token_generated => 'New API Token generated';

  @override
  String get new_trail => 'Nuovo percorso';

  @override
  String get trail_source_planner => 'Open trail planner';

  @override
  String get trail_source_record => 'Record trail';

  @override
  String get trail_source_import => 'Import file';

  @override
  String get trail_source_import_error => 'Could not import file';

  @override
  String get travel_profile_hike => 'Hike';

  @override
  String get travel_profile_hike_description =>
      'Plan a route for walking or hiking.';

  @override
  String get travel_profile_bike => 'Bike';

  @override
  String get travel_profile_bike_description => 'Plan a route for cycling.';

  @override
  String get coming_soon => 'Coming soon';

  @override
  String get no_account => 'Non hai ancora un account?';

  @override
  String get no_api_tokens => 'You have no API Tokens';

  @override
  String get no_comments_so_far => 'Nessun commento per il momento';

  @override
  String get no_data => 'Nessun dato';

  @override
  String get no_description_for_now => 'Nessuna descrizione per il momento';

  @override
  String get no_gps_data_in_image => 'No GPS data in image';

  @override
  String get no_grid => 'Nessuna griglia';

  @override
  String get no_notifications => 'Nessuna notifica';

  @override
  String get no_photos_here => 'Nessuna foto qui';

  @override
  String get no_preference => 'Nessuna preferenza';

  @override
  String get no_results => 'Nessun risultato trovato';

  @override
  String get no_routes_added => 'Nessun percorso aggiunto';

  @override
  String get no_trails_found => 'No trails found';

  @override
  String get no_waypoints_yet => 'Ancora nessun punto d\'interesse';

  @override
  String get norwegian => 'Norwegian';

  @override
  String get not_a_valid_email_address => 'Indirizzo email non valido';

  @override
  String get not_a_valid_url => 'Not a valid URL';

  @override
  String get not_completed => 'Non completato';

  @override
  String notification_comment_mention(Object user) {
    return '$user mentioned you in a comment';
  }

  @override
  String notification_list_create(Object user) {
    return '$user ha creato una nuova lista';
  }

  @override
  String notification_list_share(Object user) {
    return '$user ha condiviso una lista con te';
  }

  @override
  String get notification_new_follower => 'Hai un nuovo seguace';

  @override
  String notification_summit_log_create(Object trail, Object user) {
    return '$user created a summit log on your trail \"$trail\"';
  }

  @override
  String notification_summit_log_mention(Object user) {
    return '$user mentioned you in a summit log';
  }

  @override
  String notification_trail_comment(Object trail, Object user) {
    return '$user ha lasciato un commento sul tuo percorso \"$trail\"';
  }

  @override
  String notification_trail_create(Object user) {
    return '$user ha creato un nuovo percorso';
  }

  @override
  String notification_trail_like(Object trail, Object user) {
    return '$user liked your trail \"$trail\"';
  }

  @override
  String notification_trail_mention(Object user) {
    return '$user mentioned you in a trail';
  }

  @override
  String notification_trail_share(Object user) {
    return '$user ha condiviso un percorso con te';
  }

  @override
  String get notifications => 'Notifiche';

  @override
  String object_share_error(Object object) {
    return 'A $object must be public to be shared across instances.';
  }

  @override
  String get off => 'Spento';

  @override
  String get only_me => 'Solo io';

  @override
  String get open_in_new_tab => 'Open in new tab';

  @override
  String get or => 'o';

  @override
  String get orientation => 'Orientamento';

  @override
  String get paper_size => 'Formato carta';

  @override
  String get paragraph => 'Paragraph';

  @override
  String get parking => 'Parking';

  @override
  String get pause => 'Pause';

  @override
  String get password => 'Password';

  @override
  String get password_confirm => 'Confermare password';

  @override
  String get password_reset_sent =>
      'Un email per ripristinare la password è stato inviato';

  @override
  String get password_reset_text =>
      'Invieremo un link per ripristinare la password al tuo indirizzo email.';

  @override
  String get password_updated => 'Password aggiornata';

  @override
  String get passwords_must_match => 'Le password devono coincidere';

  @override
  String get photos => 'Foto';

  @override
  String photos_skipped_no_gps(num count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count photos skipped — no GPS data',
      one: '1 photo skipped — no GPS data',
    );
    return '$_temp0';
  }

  @override
  String get pick_a_trail => 'Scegli un percorso';

  @override
  String get planned_a_trail => 'percorso pianificato';

  @override
  String get planned_tours => 'Planned tours';

  @override
  String get pois => 'POIs';

  @override
  String get polish => 'Polacco';

  @override
  String get portuguese => 'Portoghese';

  @override
  String get print => 'Stampa';

  @override
  String get privacy => 'Privacy';

  @override
  String get private => 'Privato';

  @override
  String get profile => 'Profilo';

  @override
  String get public => 'Pubblico';

  @override
  String get public_access => 'Public access';

  @override
  String get public_share_everyone =>
      'Everyone on the internet with the link can see this trail';

  @override
  String get public_share_limited =>
      'Only people with access can open the link';

  @override
  String get public_transport => 'Public transport';

  @override
  String get radius => 'Raggio';

  @override
  String get railway_station => 'Railway station';

  @override
  String get reached_end_of_trail => 'You\'ve reached the end of the trail.';

  @override
  String get read_more => 'Per saperne di più';

  @override
  String get ready_to_join => 'Ready to join';

  @override
  String get recalculate_elevation_data => 'Recalculate elevation data';

  @override
  String get recalculating_elevation_data_hint =>
      'Recalculating elevation data will erase the existing elevation data, if any, and replace it with data from Valhalla.';

  @override
  String get register => 'Registrati';

  @override
  String get remote_users_cannot_edit => 'Remote users cannot edit';

  @override
  String get removed_trail_from => 'Percorso rimosso da';

  @override
  String get removed_trails_from => 'Percorsi rimosso da';

  @override
  String get required => 'Obbligatorio';

  @override
  String get reorder_photos_hint => 'Long-press and drag to reorder photos.';

  @override
  String get reset => 'Reset';

  @override
  String get reset_password => 'Ripristinare Password';

  @override
  String get resume => 'Resume';

  @override
  String resume_navigation_prompt(String trail) {
    return 'Resume navigation on $trail?';
  }

  @override
  String get resume_recording_prompt => 'Riprendere la registrazione?';

  @override
  String get reverse_direction => 'Reverse direction';

  @override
  String get road => 'Road';

  @override
  String route(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Routes',
      one: 'Route',
    );
    return '$_temp0';
  }

  @override
  String get route_point => 'Route Point';

  @override
  String get russian => 'Russian';

  @override
  String get save => 'Salva';

  @override
  String get save_list => 'Salta Lista';

  @override
  String get save_track => 'Save track';

  @override
  String get save_trail => 'Salva percorso';

  @override
  String get save_your_trail_first => 'Salva prima il tuo percorso';

  @override
  String get search => 'Search';

  @override
  String get search_cities => 'Cerca città';

  @override
  String get search_for_trails_places => 'Cerca percorsi, luoghi';

  @override
  String get search_list => 'Search list';

  @override
  String get search_places => 'Search places';

  @override
  String get search_trails => 'Cerca percorsi';

  @override
  String get select_date => 'Select date';

  @override
  String get select_list => 'Seleziona lista';

  @override
  String get selected => 'selected';

  @override
  String get send_to => 'Send to...';

  @override
  String get set_private => 'Set private';

  @override
  String get set_public => 'Set public';

  @override
  String get settings => 'Impostazioni';

  @override
  String get settings_notification_comment_mention =>
      'Someone mentioned you in a comment';

  @override
  String get settings_notification_list_share =>
      'Qualcuno ha condiviso una lista con te';

  @override
  String get settings_notification_new_follower => 'Hai un nuovo seguace';

  @override
  String get settings_notification_summit_log_create =>
      'Someone created a summit log on your trail';

  @override
  String get settings_notification_summit_log_mention =>
      'Someone mentioned you in a summit log';

  @override
  String get settings_notification_trail_comment =>
      'Qualcuno ha lasciato un commento sul tuo percorso';

  @override
  String get settings_notification_trail_like => 'Somone liked your trail';

  @override
  String get settings_notification_trail_mention =>
      'Someone mentioned you in a trail';

  @override
  String get settings_notification_trail_share =>
      'Qualcuno ha condiviso un percorso con te';

  @override
  String get settings_privacy_account_private =>
      'Solo tu puoi vedere il tuo profilo. Non apparirai nei risultati della ricerca. Altri utenti non potranno seguirti o condividere i percorsi con te. Tuttavia potrai comunque pubblicare percorsi o liste.';

  @override
  String get settings_privacy_account_public =>
      'Tutti possono vedere il tuo profilo. Appare nei risultati della ricerca. Altri utenti possono seguirti e condividere con te i percorsi.';

  @override
  String get settings_privacy_lists_private =>
      'Le tue liste sono private per impostazione predefinita. Nessuno tranne tu potrà vederle. Puoi cambiare queste impostazione in qualsiasi momento per liste specifiche.';

  @override
  String get settings_privacy_lists_public =>
      'Le tue liste sono pubbliche per impostazione predefinita. Tutto potrànno vederle. Puoi cambiare queste impostazione in qualsiasi momento per liste specifiche.';

  @override
  String get settings_privacy_trails_private =>
      'Le tuoi percorsi sono privati per impostazione predefinita. Nessuno tranne tu potrà vederli. Puoi cambiare queste impostazione in qualsiasi momento per percorsi specifici.';

  @override
  String get settings_privacy_trails_public =>
      'Le tuoi percorsi sono pubblici per impostazione predefinita. Tutti potranno vederli. Puoi cambiare queste impostazione in qualsiasi momento per percorsi specifici.';

  @override
  String get settings_saved => 'Settings saved';

  @override
  String get share => 'Condividi';

  @override
  String get share_profile => 'Condividere profilo';

  @override
  String get share_this_list => 'Condividi questa lista';

  @override
  String get share_this_trail => 'Condividi questo percorso';

  @override
  String get shared => 'Condiviso';

  @override
  String get shared_by => 'Condiviso da';

  @override
  String get shared_with => 'Condiviso con';

  @override
  String get shelter => 'Shelter';

  @override
  String get shortest => 'shortest';

  @override
  String get show_in_overview => 'Mostra nella panoramica';

  @override
  String get show_less => 'Show less';

  @override
  String get show_on_map => 'Mostra sulla mappa';

  @override
  String get shower => 'Shower';

  @override
  String get skiing => 'Skiing';

  @override
  String get slogan => 'Salva le tue avventure!';

  @override
  String get slope => 'Pendenza';

  @override
  String get someone => 'Qualcuno';

  @override
  String get sort => 'Ordina';

  @override
  String get spanish => 'spagnolo';

  @override
  String get speed => 'Velocità';

  @override
  String get start => 'Start';

  @override
  String get statistics => 'Statistiche';

  @override
  String get stop_drawing => 'Smettere di disegnare';

  @override
  String get stop_editing => 'Stop editing';

  @override
  String get strava_integration_after_date_hint =>
      'If your account has a large amount of acitivities you may run into Strava\'s API rate limit preventing you from syncing all activities at once. To mitigate this issue you can set an \"After\" date below so that only activities that were recorded after this date are synced.';

  @override
  String get subcategories => 'Sottocategorie';

  @override
  String get subway_stop => 'Subway entrance';

  @override
  String get summit => 'Summit';

  @override
  String get summit_book => 'Libro di vetta';

  @override
  String get table => 'Tavolo';

  @override
  String get tags => 'Tags';

  @override
  String get text => 'Testo';

  @override
  String get theme_dark => 'Dark';

  @override
  String get theme_light => 'Light';

  @override
  String get theme_system => 'Follow system';

  @override
  String get tilesets => 'Riquadri personalizzati';

  @override
  String get time => 'Time';

  @override
  String get time_in_motion => 'Time in Motion';

  @override
  String get toilets => 'Toilets';

  @override
  String get top_speed => 'Top Speed';

  @override
  String get tourism => 'Tourism';

  @override
  String trail(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Percorsi',
      one: 'Percorso',
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
  String get trail_not_shared => 'Percorso non condiviso con nessuno';

  @override
  String get trail_saved_successfully => 'Percorso salvato con successo';

  @override
  String get some_waypoints_failed_to_save =>
      'Trail saved, but some waypoints failed to save';

  @override
  String get trails_for_you => 'Percorsi per te';

  @override
  String get tram_stop => 'Tram stop';

  @override
  String get unchanged => 'unchanged';

  @override
  String get units => 'Unità';

  @override
  String get unlink => 'Unlink';

  @override
  String get upload_file => 'Carica file';

  @override
  String get upload_gpx => 'Carica file GPX';

  @override
  String get upload_new_file => 'Upload new file';

  @override
  String get uploaded => 'caricato';

  @override
  String get uploaded_trail_to_hammerhead =>
      'Successfully uploaded trail to Hammerhead';

  @override
  String get use_hills => 'Use hills';

  @override
  String get use_roads => 'Use Roads';

  @override
  String get users => 'Users';

  @override
  String get username => 'Nome utente';

  @override
  String get username_not_unique =>
      'This username is already taken. Please try another.';

  @override
  String get view => 'Visualizza';

  @override
  String get viewpoint => 'Viewpoint';

  @override
  String get visibilty => 'Visibility';

  @override
  String get visibilty_status => 'Visibility status';

  @override
  String get walking_speed => 'Walking speed';

  @override
  String get water => 'Water';

  @override
  String get web => 'Web';

  @override
  String waypoints(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Punti di passaggio',
      one: 'Punto di passaggio',
    );
    return '$_temp0';
  }

  @override
  String get welcome_to => 'Welcome to';

  @override
  String get width => 'Width';

  @override
  String get wrong_username_or_password => 'Nome utente o password errati';

  @override
  String get you_can => 'Puoi';

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

  @override
  String get settings_categories_reorder_hint =>
      'Categories control which trail types you see and in what order. Turn one off to hide it as a filter — your trails stay published, they just won\'t appear under that category. Tap a category to manage its subcategories individually.\n\nTo change the order, press and hold a row, then drag it to a new position. The order you set here is reflected everywhere categories are shown.';
}
