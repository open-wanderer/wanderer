// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Catalan Valencian (`ca`).
class AppLocalizationsCa extends AppLocalizations {
  AppLocalizationsCa([String locale = 'ca']) : super(locale);

  @override
  String get about => 'Quant a';

  @override
  String get about_changelog => 'Novetats';

  @override
  String get about_discord_subtitle => 'Xateja amb la comunitat';

  @override
  String get about_github_subtitle => 'Codi font, incidències i contribucions';

  @override
  String get about_license => 'Llicència';

  @override
  String get about_licenses => 'Llicències de codi obert';

  @override
  String get about_privacy_policy => 'Política de privadesa';

  @override
  String get about_report_bug => 'Informa d\'un error';

  @override
  String about_version(String version) {
    return 'Versió $version';
  }

  @override
  String get appearance => 'Aparença';

  @override
  String get account => 'Compte';

  @override
  String get account_delete_confirm =>
      'Esteu a punt d\'eliminar el vostre compte. També s\'eliminaran tots els vostres senders. Voleu continuar?';

  @override
  String get account_privacy => 'Privadesa del compte';

  @override
  String get add_bio => 'Afegeix biografia';

  @override
  String get add_waypoint => 'Afegeix punt de ruta';

  @override
  String get after => 'Després de';

  @override
  String get all => 'Tot';

  @override
  String get altitude => 'Altitud';

  @override
  String get author => 'Autor';

  @override
  String get average_speed => 'Velocitat mitjana';

  @override
  String get background_location_body =>
      'wanderer recull dades d\'ubicació en segon pla perquè el teu sender es continuï enregistrant quan la pantalla està apagada o l\'aplicació està tancada. El traçat enregistrat es queda al teu dispositiu fins que decideixis desar el sender.\n\nAndroid només ofereix aquesta opció a la configuració del sistema: obre Permisos → Ubicació i tria «Permet-ho sempre».';

  @override
  String get background_location_confirm => 'Obre la configuració';

  @override
  String get background_location_title => 'Continua enregistrant en segon pla';

  @override
  String get basic_info => 'Informació bàsica';

  @override
  String get before => 'Abans de';

  @override
  String get by => 'per';

  @override
  String get cancel => 'Cancel·la';

  @override
  String get discard => 'Descarta';

  @override
  String get discard_trail_confirm =>
      'Voleu descartar aquest sender i els seus canvis?';

  @override
  String card(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Targetes',
      one: 'Targeta',
    );
    return '$_temp0';
  }

  @override
  String get categories => 'Categories';

  @override
  String get category => 'Categoria';

  @override
  String get change_email => 'Canvia el correu electrònic';

  @override
  String get change_password => 'Canvia la contrasenya';

  @override
  String get clear_all => 'Esborra-ho tot';

  @override
  String get close => 'Tanca';

  @override
  String comment(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Comentaris',
      one: 'Comentari',
    );
    return '$_temp0';
  }

  @override
  String get completed => 'Completat';

  @override
  String get completion_status => 'Estat de finalització';

  @override
  String get confirm_deletion => 'Confirma l\'eliminació';

  @override
  String get couldnt_start_navigation =>
      'No s\'ha pogut iniciar la navegació. Comprova la connexió i torna-ho a provar.';

  @override
  String get location_services_disabled =>
      'Els serveis d\'ubicació estan desactivats. Activa el GPS per utilitzar la navegació.';

  @override
  String get location_permission_denied =>
      'Cal el permís d\'ubicació per a la navegació.';

  @override
  String get location_permission_permanently_denied =>
      'El permís d\'ubicació està denegat permanentment. Activa\'l a la configuració.';

  @override
  String get location_unavailable =>
      'No s\'ha pogut determinar la teva ubicació. Torna-ho a provar.';

  @override
  String get copy_link => 'Copia l\'enllaç';

  @override
  String get create_waypoint => 'Crea un punt de ruta';

  @override
  String get creation_date => 'Data de creació';

  @override
  String get current_password => 'Contrasenya actual';

  @override
  String get danger_zone => 'Zona de perill';

  @override
  String get date => 'Data';

  @override
  String get delete => 'Elimina';

  @override
  String get not_now => 'Ara no';

  @override
  String get open => 'Obre';

  @override
  String get delete_account => 'Elimina el compte';

  @override
  String get delete_trail_confirm =>
      'Segur que voleu eliminar aquest sender? Aquesta acció no es pot desfer.';

  @override
  String get delete_blocked_while_uploading =>
      'Aquest sender s\'està carregant ara mateix. Espera que acabi la pujada i torna-ho a provar.';

  @override
  String get delete_unsynced_trail_confirm =>
      'Voleu eliminar aquest sender? Encara no s\'ha pujat, així que això no es pot desfer.';

  @override
  String get delete_needs_connection =>
      'Aquest sender ja és al servidor. Connecteu-vos a internet per eliminar-lo.';

  @override
  String get description => 'Descripció';

  @override
  String get difficult => 'Difícil';

  @override
  String get difficulty => 'Dificultat';

  @override
  String get directions => 'Indicacions';

  @override
  String get distance => 'Distància';

  @override
  String get download => 'Baixa';

  @override
  String get duration => 'Durada';

  @override
  String get easy => 'Fàcil';

  @override
  String get edit => 'Edita';

  @override
  String get edit_needs_connection =>
      'L\'edició funciona sobre la còpia del sender que hi ha al servidor. Connecteu-vos a internet per editar-lo.';

  @override
  String get edit_waypoint => 'Edita el punt de ruta';

  @override
  String get edited => 'editat';

  @override
  String get elevation_gain => 'Desnivell positiu';

  @override
  String get elevation_loss => 'Desnivell negatiu';

  @override
  String get elevation_profile => 'Perfil d\'elevació';

  @override
  String get email => 'Correu electrònic';

  @override
  String get email_not_unique =>
      'Aquesta adreça de correu electrònic ja està en ús.';

  @override
  String get email_updated => 'Correu electrònic actualitzat';

  @override
  String get error_deleting_trail => 'Error en eliminar el sender';

  @override
  String get error_reading_file => 'Error en llegir el fitxer';

  @override
  String get error_saving_settings => 'Error en desar la configuració';

  @override
  String get error_saving_trail => 'Error en desar el sender';

  @override
  String get error_updating_password => 'Error en actualitzar la contrasenya';

  @override
  String get explore => 'Explora';

  @override
  String get exit_navigation => 'Surt';

  @override
  String get stop_navigation_confirm =>
      'Voleu aturar la navegació i tornar al sender?';

  @override
  String get stop_recording => 'Atura l\'enregistrament';

  @override
  String get stop_recording_confirm => 'Voleu aturar l\'enregistrament?';

  @override
  String get search_this_area => 'Cerca en aquesta zona';

  @override
  String get filter_tags => 'Filtra les etiquetes';

  @override
  String get filter_trails => 'Filtra els senders';

  @override
  String get finish => 'Final';

  @override
  String get finish_disabled_hint =>
      'Afegeix com a mínim 2 àncores per acabar la ruta.';

  @override
  String get follow => 'Segueix';

  @override
  String get followers => 'Seguidors';

  @override
  String get following => 'Seguint';

  @override
  String get from_photos => 'Des de fotos';

  @override
  String get help => 'Ajuda';

  @override
  String get hiking => 'Senderisme';

  @override
  String get home => 'Inici';

  @override
  String get icon => 'Icona';

  @override
  String get imperial => 'Imperial';

  @override
  String get joined => 'Es va unir';

  @override
  String get language => 'Idioma';

  @override
  String get latitude => 'Latitud';

  @override
  String get like_status => 'Estat del m\'agrada';

  @override
  String get liked => 'Agradat';

  @override
  String get likes => 'M\'agrada';

  @override
  String list(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Llistes',
      one: 'Llista',
    );
    return '$_temp0';
  }

  @override
  String get location => 'Ubicació';

  @override
  String get locations => 'Ubicacions';

  @override
  String get center_on_my_location => 'Centra a la meva ubicació';

  @override
  String get location_tracking_notification_title => 'wanderer';

  @override
  String get location_tracking_notification_text =>
      'S\'està enregistrant el teu sender';

  @override
  String location_tracking_notification_text_navigating(String trail) {
    return 'Navegant per $trail';
  }

  @override
  String get login => 'Inicia sessió';

  @override
  String get logout => 'Tanca la sessió';

  @override
  String get longitude => 'Longitud';

  @override
  String get map => 'Mapa';

  @override
  String get metric => 'Mètric';

  @override
  String get moderate => 'Moderat';

  @override
  String get my_account => 'El meu compte';

  @override
  String in_distance(String distance) {
    return 'en $distance';
  }

  @override
  String get name => 'Nom';

  @override
  String get navigate => 'Navega';

  @override
  String get new_password => 'Contrasenya nova';

  @override
  String get new_password_success => 'S\'ha establert la contrasenya nova';

  @override
  String get new_trail => 'Sender nou';

  @override
  String get trail_source_planner => 'Obre el planificador de rutes';

  @override
  String get trail_source_planner_description =>
      'Dibuixa una ruta nova al mapa, punt de ruta a punt de ruta.';

  @override
  String get trail_source_record => 'Enregistra un sender';

  @override
  String get trail_source_record_description =>
      'Segueix les teves coordenades en directe i registra el teu recorregut en temps real.';

  @override
  String get trail_source_import => 'Importa un fitxer';

  @override
  String get trail_source_import_description =>
      'Puja fitxers GPX, KML, KMZ, TCX o FIT directament des de l\'emmagatzematge del teu dispositiu.';

  @override
  String get trail_source_import_error => 'No s\'ha pogut importar el fitxer';

  @override
  String get trail_source_offline_import_error =>
      'Només es poden importar fitxers GPX sense connexió';

  @override
  String get no_comments_so_far => 'Encara no hi ha comentaris';

  @override
  String get no_data => 'Sense dades';

  @override
  String get no_description_for_now => 'Encara no hi ha descripció';

  @override
  String get no_gps_data_in_image =>
      'No s\'han trobat dades GPS en aquesta foto.';

  @override
  String get no_preference => 'Sense preferència';

  @override
  String get no_trails_found => 'No s\'ha trobat cap sender';

  @override
  String get not_completed => 'No completat';

  @override
  String get notifications => 'Notificacions';

  @override
  String get only_me => 'Només jo';

  @override
  String get or => 'o';

  @override
  String get own_trails_empty_body =>
      'Els senders que enregistris o desis sense connexió apareixeran aquí, i es pujaran automàticament quan tornis a estar en línia.';

  @override
  String get own_trails_empty_title => 'Encara no hi ha res desat';

  @override
  String get own_trails_offline_banner =>
      'Sense connexió: només es mostren els senders d\'aquest dispositiu.';

  @override
  String get trails_on_device => 'Senders (al dispositiu)';

  @override
  String get pause => 'Pausa';

  @override
  String get password => 'Contrasenya';

  @override
  String get password_confirm => 'Confirma la contrasenya';

  @override
  String get passwords_must_match => 'Les contrasenyes han de coincidir';

  @override
  String photo_copy_failed_toast(num count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: 'Sender desat, però $count fotos no s\'han pogut desar.',
      one: 'Sender desat, però 1 foto no s\'ha pogut desar.',
    );
    return '$_temp0';
  }

  @override
  String get photos => 'Fotos i vídeos';

  @override
  String photos_skipped_no_gps(num count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count fotos omeses: no tenen dades GPS',
      one: '1 foto omesa: no té dades GPS',
    );
    return '$_temp0';
  }

  @override
  String get privacy => 'Privadesa';

  @override
  String get private => 'Privat';

  @override
  String get profile => 'Perfil';

  @override
  String get public => 'Públic';

  @override
  String get reached_end_of_trail => 'Has arribat al final del sender.';

  @override
  String get register => 'Registra\'t';

  @override
  String get reorder_photos_hint =>
      'Mantén premut i arrossega per reordenar les fotos.';

  @override
  String get reset => 'Restableix';

  @override
  String get resume => 'Reprèn';

  @override
  String resume_navigation_prompt(String trail) {
    return 'Voleu reprendre la navegació a $trail?';
  }

  @override
  String get resume_recording_prompt => 'Voleu reprendre l\'enregistrament?';

  @override
  String route(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Rutes',
      one: 'Ruta',
    );
    return '$_temp0';
  }

  @override
  String get follow_roads => 'Segueix les carreteres';

  @override
  String get follow_roads_description =>
      'Ajusta el traçat enregistrat a les carreteres i camins més propers.';

  @override
  String get recalculate_heights => 'Recalcula les altituds';

  @override
  String get recalculate_heights_description =>
      'Substitueix l\'elevació enregistrada pel GPS per valors més precisos del mapa.';

  @override
  String get track_transform_failed =>
      'Couldn\'t adjust the track. Check your connection and try again.';

  @override
  String get save => 'Desa';

  @override
  String get save_track => 'Desa el traçat';

  @override
  String get search => 'Cerca';

  @override
  String get search_for_trails_places => 'Cerca senders, llistes, llocs';

  @override
  String get select_date => 'Selecciona una data';

  @override
  String get settings => 'Configuració';

  @override
  String get settings_notification_comment_mention =>
      'Algú t\'ha mencionat en un comentari';

  @override
  String get settings_notification_list_share =>
      'Algú ha compartit una llista amb tu';

  @override
  String get settings_notification_new_follower => 'Tens un seguidor nou';

  @override
  String get settings_notification_summit_log_create =>
      'Algú ha creat un llibre de cims al teu sender';

  @override
  String get settings_notification_summit_log_mention =>
      'Algú t\'ha mencionat en un llibre de cims';

  @override
  String get settings_notification_trail_comment =>
      'Algú ha deixat un comentari al teu sender';

  @override
  String get settings_notification_trail_like =>
      'A algú li agrada el teu sender';

  @override
  String get settings_notification_trail_mention =>
      'Algú t\'ha mencionat en un sender';

  @override
  String get settings_notification_trail_share =>
      'Algú ha compartit un sender amb tu';

  @override
  String get settings_privacy_account_private =>
      'Només tu pots veure el teu perfil. No apareixeràs als resultats de cerca. Els altres usuaris no et poden seguir ni compartir senders amb tu. Encara pots publicar senders o llistes.';

  @override
  String get settings_privacy_account_public =>
      'Tothom pot veure el teu perfil. Apareixes als resultats de cerca. Els altres usuaris et poden seguir i compartir senders amb tu.';

  @override
  String get settings_privacy_lists_private =>
      'Les teves llistes són privades per defecte. Ningú excepte tu no les podrà veure. Pots canviar aquesta configuració en qualsevol moment per a llistes individuals.';

  @override
  String get settings_privacy_lists_public =>
      'Les teves llistes són públiques per defecte. Tothom les podrà veure. Pots canviar aquesta configuració en qualsevol moment per a llistes individuals.';

  @override
  String get settings_privacy_trails_private =>
      'Els teus senders són privats per defecte. Ningú excepte tu no els podrà veure. Pots canviar aquesta configuració en qualsevol moment per a senders individuals.';

  @override
  String get settings_privacy_trails_public =>
      'Els teus senders són públics per defecte. Tothom els podrà veure. Pots canviar aquesta configuració en qualsevol moment per a senders individuals.';

  @override
  String get share => 'Comparteix';

  @override
  String get share_profile => 'Comparteix el perfil';

  @override
  String get shared => 'Compartit';

  @override
  String get show_on_map => 'Mostra al mapa';

  @override
  String signout_unsynced_warning(num count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other:
          'Tens $count senders que encara no s\'han pujat. Tancar la sessió no els eliminarà — seran aquí quan tornis a iniciar sessió —, però no es pujaran fins llavors.',
      one: 'Tens 1 sender que encara no s\'ha pujat. Tancar la sessió no l\'eliminarà — serà aquí quan tornis a iniciar sessió —, però no es pujarà fins llavors.',
    );
    return '$_temp0';
  }

  @override
  String get slogan => 'Els teus senders. Les teves dades. El teu servidor.';

  @override
  String get sort => 'Ordena';

  @override
  String get speed => 'Velocitat';

  @override
  String get start => 'Inici';

  @override
  String get subcategories => 'Subcategories';

  @override
  String get summit_book => 'Llibre de cims';

  @override
  String get sync_discard_draft_confirm =>
      'Voleu eliminar aquest enregistrament? No s\'ha pujat enlloc.';

  @override
  String get sync_draft => 'Esborrany';

  @override
  String get sync_failed => 'La pujada ha fallat · Toca per tornar-ho a provar';

  @override
  String get sync_pending => 'Pendent de pujar';

  @override
  String get sync_section_failed_body =>
      'La pujada no s\'ha completat. Torna-ho a provar.';

  @override
  String get sync_section_pending_online_body =>
      'Es pujarà automàticament d\'aquí a un moment.';

  @override
  String get sync_section_uploading_body =>
      'S\'està enviant aquest sender al teu servidor.';

  @override
  String get sync_section_draft_body =>
      'Aquest enregistrament només és en aquest dispositiu.';

  @override
  String get sync_section_draft_title => 'Encara no s\'ha pujat';

  @override
  String get sync_section_failed_title => 'La pujada ha fallat';

  @override
  String get sync_section_pending_offline_body =>
      'Es pujarà quan tornis a estar en línia';

  @override
  String get sync_section_retry => 'Torna-ho a provar';

  @override
  String get sync_section_upload => 'Puja';

  @override
  String get sync_section_upload_now => 'Puja-ho ara';

  @override
  String get sync_uploading => 'S\'està pujant…';

  @override
  String get tags => 'Etiquetes';

  @override
  String get theme_dark => 'Fosc';

  @override
  String get theme_light => 'Clar';

  @override
  String get theme_system => 'Segueix el sistema';

  @override
  String get time_in_motion => 'Temps en moviment';

  @override
  String trail(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Senders',
      one: 'Sender',
    );
    return '$_temp0';
  }

  @override
  String get trail_saved_successfully => 'Sender desat correctament';

  @override
  String get trail_not_on_this_device =>
      'Aquest sender ja no és en aquest dispositiu.';

  @override
  String get trail_uploaded_reopen_to_edit =>
      'Aquest sender s\'ha acabat de pujar. Torna\'l a obrir des dels teus senders per continuar editant-lo.';

  @override
  String get some_waypoints_failed_to_save =>
      'Sender desat, però alguns punts de ruta no s\'han pogut desar';

  @override
  String get units => 'Unitats';

  @override
  String get users => 'Usuaris';

  @override
  String get username => 'Nom d\'usuari';

  @override
  String get username_not_unique =>
      'Aquest nom d\'usuari ja està en ús. Prova\'n un altre.';

  @override
  String get visibilty_status => 'Estat de visibilitat';

  @override
  String get web => 'Web';

  @override
  String waypoints(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Punts de ruta',
      one: 'Punt de ruta',
    );
    return '$_temp0';
  }

  @override
  String get welcome_to => 'Benvingut a';

  @override
  String get wrong_username_or_password =>
      'Nom d\'usuari o contrasenya incorrectes';

  @override
  String get server_incompatible_title => 'Servidor no compatible';

  @override
  String get server_incompatible =>
      'Aquest servidor executa una versió de wanderer que l\'aplicació no admet. Iniciar sessió serà possible un cop s\'actualitzi.';

  @override
  String server_incompatible_version(String version) {
    return 'Versió del servidor: $version';
  }

  @override
  String get server_incompatible_docs => 'Com actualitzar el servidor';

  @override
  String get you_have_arrived => 'Has arribat';

  @override
  String get settings_categories_confirm_disable_title =>
      'Voleu amagar aquesta categoria?';

  @override
  String get settings_categories_confirm_disable_subcategory_title =>
      'Voleu amagar aquesta subcategoria?';

  @override
  String settings_categories_confirm_disable_body(int count) {
    return '$count dels teus senders utilitzen aquesta categoria. Continuaran publicats, però aquest filtre quedarà amagat.';
  }

  @override
  String get settings_categories_confirm_view_trails => 'Mostra els senders';

  @override
  String get settings_categories_confirm_disable_confirm =>
      'Desactiva-ho de totes maneres';

  @override
  String get settings_categories_empty_title => 'Sense subcategories';

  @override
  String get settings_categories_empty_body =>
      'Aquesta categoria no té subcategories per configurar.';

  @override
  String get settings_categories_reorder_hint =>
      'Les categories controlen quins tipus de senders veus i en quin ordre. Desactiva\'n una per amagar-la com a filtre: els teus senders continuaran publicats, simplement no apareixeran sota aquesta categoria. Toca una categoria per gestionar-ne les subcategories individualment.\n\nPer canviar l\'ordre, mantén premuda una fila i arrossega-la a una posició nova. L\'ordre que estableixis aquí es reflecteix a tots els llocs on es mostren categories.';

  @override
  String get something_went_wrong => 'Alguna cosa ha anat malament';

  @override
  String get technical_details => 'Detalls tècnics';

  @override
  String get link => 'Enllaç';

  @override
  String get url => 'URL';

  @override
  String get open_in_new_tab => 'Obre en una pestanya nova';

  @override
  String get remove => 'Elimina';

  @override
  String get remove_download_confirm_body =>
      'Això elimina la còpia baixada d\'aquest dispositiu. El sender en si no s\'elimina: hauràs de tornar-lo a baixar per utilitzar-lo sense connexió.';

  @override
  String get apply => 'Aplica';

  @override
  String get add_at_least_2_anchors_hint =>
      'Afegeix com a mínim 2 àncores per veure el perfil d\'elevació.';

  @override
  String get reverse_direction => 'Inverteix la direcció';

  @override
  String get delete_all => 'Elimina-ho tot';

  @override
  String get auto_routing => 'Encaminament automàtic';

  @override
  String get auto_routing_hint =>
      'Segueix automàticament carreteres i camins entre les àncores.';

  @override
  String get travel_profile => 'Perfil de desplaçament';

  @override
  String get no_track_data => 'Sense dades de traçat';

  @override
  String get available_offline => 'Disponible sense connexió';

  @override
  String get no_lists_found => 'No s\'ha trobat cap llista';

  @override
  String get search_lists => 'Cerca llistes…';

  @override
  String get search_for_a_location => 'Cerca una ubicació';

  @override
  String no_results_for_query(String query) {
    return 'No hi ha resultats per a «$query»';
  }

  @override
  String get filter => 'Filtra';

  @override
  String no_label_yet(String label) {
    return 'Encara no hi ha cap $label.';
  }

  @override
  String get no_lists_yet => 'Encara no hi ha llistes.';

  @override
  String get no_bio_yet => 'Encara no hi ha biografia.';

  @override
  String get show_more => 'Mostra\'n més';

  @override
  String get show_less => 'Mostra\'n menys';

  @override
  String get feed => 'Canal';

  @override
  String get profile_feed_empty_own => 'Encara no tens cap activitat.';

  @override
  String profile_feed_empty_other(String username) {
    return '$username encara no té cap activitat.';
  }

  @override
  String get no_trails_yet => 'Encara no hi ha senders.';

  @override
  String get library_empty_title => 'No hi ha senders baixats';

  @override
  String get library_empty_body =>
      'Els senders que baixis es guarden aquí perquè els puguis obrir sense connexió.';

  @override
  String get library_empty_search_body =>
      'Prova un altre terme de cerca o esborra els filtres.';

  @override
  String get search_location => 'Cerca una ubicació';

  @override
  String no_servers_match_query(String query) {
    return 'No hi ha cap servidor que coincideixi amb «$query»';
  }

  @override
  String get use_custom_url_instead => 'Utilitza una URL personalitzada';

  @override
  String get select_instance => 'Selecciona una instància';

  @override
  String get enter_server_url_hint =>
      'Introdueix l\'URL del servidor (p. ex. wanderer.to)';

  @override
  String get last_used => 'Últim ús';

  @override
  String get community_instances => 'Instàncies de la comunitat';

  @override
  String get search_library => 'Cerca a la biblioteca…';

  @override
  String language_and_units(String language, String units) {
    return '$language i $units';
  }

  @override
  String get edit_route => 'Edita la ruta';

  @override
  String get undo => 'Desfés';

  @override
  String get redo => 'Refés';

  @override
  String get bold => 'Negreta';

  @override
  String get italic => 'Cursiva';

  @override
  String get underline => 'Subratllat';

  @override
  String get bullet_list => 'Llista amb pics';

  @override
  String get ordered_list => 'Llista numerada';

  @override
  String get blockquote => 'Citació';

  @override
  String get library => 'Biblioteca';

  @override
  String get settings_offline_regions_title => 'Mapes/regions sense connexió';

  @override
  String get regions_search_hint => 'Cerca regions';

  @override
  String get regions_dem_toggle_label => 'Baixa dades d\'elevació (DEM)';

  @override
  String get regions_dem_toggle_caption =>
      'Afegeix ombrejat del relleu; augmenta la mida de la baixada';

  @override
  String get regions_update_available => 'Actualització disponible';

  @override
  String get regions_update_action => 'Actualitza';

  @override
  String get regions_retry => 'Torna-ho a provar';

  @override
  String get regions_not_yet_available => 'Encara no disponible';

  @override
  String get regions_build_failed => 'La compilació ha fallat';

  @override
  String regions_delete_confirm_title(String name) {
    return 'Voleu eliminar $name?';
  }

  @override
  String get regions_delete_confirm_body =>
      'Això elimina el mapa baixat i les dades d\'elevació d\'aquesta regió. Hauràs de tornar-los a baixar per utilitzar-los sense connexió.';

  @override
  String get regions_delete_confirm_action => 'Elimina';

  @override
  String regions_disk_usage_summary(String size, num count) {
    return '$size en ús en $count regió/regions baixades';
  }

  @override
  String get regions_empty_search_title => 'No hi ha regions coincidents';

  @override
  String get regions_empty_search_body => 'Prova un altre terme de cerca.';

  @override
  String get regions_empty_catalog_title =>
      'No hi ha regions sense connexió disponibles';

  @override
  String get regions_empty_catalog_body =>
      'Demana a l\'administrador de la teva instància de Wanderer que configuri regions per baixar.';

  @override
  String get regions_vector_tile_title => 'Vector';

  @override
  String get regions_dem_tile_title => 'Dades d\'elevació';

  @override
  String get regions_download_failed => 'La baixada ha fallat';

  @override
  String get regions_dem_locked_subtitle => 'Baixa primer les dades del mapa';

  @override
  String get regions_offline_unavailable_title =>
      'No es poden carregar les regions';

  @override
  String get regions_offline_unavailable_body =>
      'Connecta\'t a internet per explorar i gestionar les regions que es poden baixar.';

  @override
  String get regions_map_geometry_failed =>
      'No s\'ha pogut carregar el contorn de la regió';

  @override
  String get regions_map_back_label => 'Torna a les regions';

  @override
  String regions_group_expand_label(String name) {
    return 'Desplega $name';
  }

  @override
  String regions_group_collapse_label(String name) {
    return 'Replega $name';
  }

  @override
  String get offline_title => 'Estàs sense connexió';

  @override
  String get offline_try_again => 'Torna-ho a provar';

  @override
  String get offline_map_body =>
      'Connecta\'t a internet per carregar el mapa. Els senders baixats encara estan disponibles.';

  @override
  String get offline_list_body =>
      'Connecta\'t a internet per carregar les llistes.';

  @override
  String get offline_profile_body =>
      'Connecta\'t a internet per carregar el teu perfil complet.';

  @override
  String get offline_settings_banner =>
      'Esteu sense connexió. La configuració és només de lectura fins que us torneu a connectar.';

  @override
  String get offline_action_unavailable =>
      'Estàs sense connexió: torna-ho a provar quan tornis a estar en línia.';

  @override
  String get offline_categories_body =>
      'Connecta\'t a internet per gestionar les categories.';

  @override
  String get offline_trail_search_body =>
      'Connecta\'t a internet per cercar senders. Els senders baixats encara estan disponibles.';

  @override
  String get selected_location => 'Selected location';

  @override
  String distance_from_your_location(String distance) {
    return '$distance from your location';
  }

  @override
  String get plan_route_from_here => 'Plan route from here';

  @override
  String get recent_searches => 'Recent searches';

  @override
  String get clear_search_history => 'Clear search history';

  @override
  String get clear_search_history_confirm => 'Remove all recent searches?';
}
