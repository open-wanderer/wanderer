// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Chinese (`zh`).
class AppLocalizationsZh extends AppLocalizations {
  AppLocalizationsZh([String locale = 'zh']) : super(locale);

  @override
  String get about => '关于';

  @override
  String get appearance => 'Appearance';

  @override
  String get account => 'Account';

  @override
  String get account_delete_confirm => '您现在要删除当前账户，所有的路线都会删除无法恢复，确认继续操作吗？';

  @override
  String get account_privacy => '账户隐私';

  @override
  String get add_bio => '添加个人信息';

  @override
  String get add_waypoint => '添加坐标';

  @override
  String get adjust_track => 'Adjust track';

  @override
  String get after => '之后';

  @override
  String get all => 'All';

  @override
  String get altitude => '海拔';

  @override
  String get author => '作者';

  @override
  String get avatar => '头像';

  @override
  String get average_speed => '平均速度';

  @override
  String get basic_info => '基本信息';

  @override
  String get before => '之前';

  @override
  String get behavior => '行为';

  @override
  String get by => '由';

  @override
  String get cancel => '取消';

  @override
  String get discard => 'Discard';

  @override
  String get discard_trail_confirm => 'Discard this trail and its changes?';

  @override
  String get car => '汽车';

  @override
  String card(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '卡片',
      one: '卡片',
    );
    return '$_temp0';
  }

  @override
  String get categories => '分类';

  @override
  String get category => '分类';

  @override
  String get change_email => '更改邮箱';

  @override
  String get change_password => '更改密码';

  @override
  String get clear_all => '清除全部';

  @override
  String get close => '关闭';

  @override
  String comment(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '评论',
      one: '评论',
    );
    return '$_temp0';
  }

  @override
  String get completed => '已完成';

  @override
  String get completion_status => '完成状态';

  @override
  String get confirm_deletion => '确认删除';

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
  String get copy_link => '复制链接';

  @override
  String get create_waypoint => '创建路径点';

  @override
  String get creation_date => '创建日期';

  @override
  String get crop => '裁剪';

  @override
  String get cross => '交叉点';

  @override
  String get current_password => '当前密码';

  @override
  String get danger_zone => '危险区域';

  @override
  String get date => '日期';

  @override
  String get delete => '删除';

  @override
  String get open => 'Open';

  @override
  String get delete_account => '注销账户';

  @override
  String get delete_trail_confirm => '您确认想要删除当前行程？此操作不可撤回。';

  @override
  String get delete_blocked_while_uploading =>
      'This trail is uploading right now. Wait for the upload to finish, then try again.';

  @override
  String get delete_unsynced_trail_confirm =>
      'Delete this trail? It hasn\'t been uploaded yet, so this can\'t be undone.';

  @override
  String get delete_needs_connection =>
      'This trail is already on the server. Connect to the internet to delete it.';

  @override
  String get description => '描述';

  @override
  String get difficult => '困难';

  @override
  String get difficulty => '难度';

  @override
  String get directions => '导航';

  @override
  String get display => '显示';

  @override
  String get distance => '距离';

  @override
  String get download => '下载';

  @override
  String get duration => '持续时间';

  @override
  String get easy => '简单';

  @override
  String get edit => '编辑';

  @override
  String get edit_needs_connection =>
      'Editing works on the server copy of this trail. Connect to the internet to edit it.';

  @override
  String get edit_waypoint => '编辑坐标';

  @override
  String get edited => '已编辑';

  @override
  String get elevation_gain => '上升海拔';

  @override
  String get elevation_loss => '下降海拔';

  @override
  String get elevation_profile => 'Elevation Profile';

  @override
  String get email => '电子邮箱';

  @override
  String get email_not_unique => '此电子邮件地址已被使用。';

  @override
  String get email_updated => '邮箱已更改';

  @override
  String get error_deleting_trail => 'Error deleting trail';

  @override
  String get error_reading_file => '读取文件错误';

  @override
  String get error_saving_settings => 'Error saving settings';

  @override
  String get error_saving_trail => '保存路线失败';

  @override
  String get error_updating_password => '更新密码失败';

  @override
  String get explore => '探索';

  @override
  String get exit_navigation => 'Exit';

  @override
  String get stop_navigation_confirm =>
      'Stop navigation and return to the trail?';

  @override
  String get stop_recording => '停止录制';

  @override
  String get stop_recording_confirm => '停止录制？';

  @override
  String get search_this_area => 'Search this area';

  @override
  String get ferry => '轮渡';

  @override
  String get filter_tags => '过滤标签';

  @override
  String get filter_trails => 'Filter trails';

  @override
  String get finish => '完成';

  @override
  String get finish_disabled_hint =>
      'Add at least 2 anchors to finish your route.';

  @override
  String get follow => '关注';

  @override
  String get followers => '关注者';

  @override
  String get following => '已关注';

  @override
  String get from_photos => '来自照片';

  @override
  String get heading => '标题';

  @override
  String get height => '高度';

  @override
  String get help => '帮助';

  @override
  String get hiking => '徒步';

  @override
  String get home => '主页';

  @override
  String get hotel => '酒店';

  @override
  String get icon => '图标';

  @override
  String get imperial => '英制';

  @override
  String get joined => '已加入';

  @override
  String get language => '语言';

  @override
  String get latitude => '纬度';

  @override
  String get like_status => '喜欢状态';

  @override
  String get liked => '赞';

  @override
  String get likes => '赞';

  @override
  String list(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '列表',
      one: '列表',
    );
    return '$_temp0';
  }

  @override
  String get location => '地点';

  @override
  String get locations => 'Locations';

  @override
  String get center_on_my_location => 'Center on my location';

  @override
  String get location_tracking_notification_title => 'wanderer';

  @override
  String get location_tracking_notification_text => 'Recording your trail';

  @override
  String location_tracking_notification_text_navigating(String trail) {
    return 'Navigating $trail';
  }

  @override
  String get login => '登录';

  @override
  String get logout => '登出';

  @override
  String get longitude => '经度';

  @override
  String get loop => '循环';

  @override
  String get map => '地图';

  @override
  String get metric => '公制';

  @override
  String get moderate => '中等';

  @override
  String get more => '更多';

  @override
  String get mountain => '山地';

  @override
  String get my_account => '我的账户';

  @override
  String in_distance(String distance) {
    return 'in $distance';
  }

  @override
  String get name => '名称';

  @override
  String get navigate => 'Navigate';

  @override
  String get near => '附近';

  @override
  String get new_password => '新密码';

  @override
  String get new_password_success => '新密码已设置';

  @override
  String get new_trail => '创建新路线';

  @override
  String get trail_source_planner => 'Open trail planner';

  @override
  String get trail_source_planner_description =>
      'Draw a new route on the map, waypoint by waypoint.';

  @override
  String get trail_source_record => 'Record trail';

  @override
  String get trail_source_record_description =>
      'Track your live coordinates and log your journey in real-time.';

  @override
  String get trail_source_import => 'Import file';

  @override
  String get trail_source_import_description =>
      'Upload GPX, KML, KMZ, TCX or FIT files directly from your device storage.';

  @override
  String get trail_source_import_error => 'Could not import file';

  @override
  String get trail_source_offline_import_error =>
      'Only GPX files can be imported offline';

  @override
  String get no_comments_so_far => '到目前为止没有评论';

  @override
  String get no_data => '无数据';

  @override
  String get no_description_for_now => '暂无描述';

  @override
  String get no_gps_data_in_image => '图像中没有GPS数据';

  @override
  String get no_preference => '尚未规划';

  @override
  String get no_trails_found => 'No trails found';

  @override
  String get not_completed => '未完成';

  @override
  String get notifications => '通知';

  @override
  String get only_me => '仅自己';

  @override
  String get or => '或';

  @override
  String get orientation => '方向';

  @override
  String get own_trails_empty_body =>
      'Trails you record or save offline appear here, and upload automatically once you\'re back online.';

  @override
  String get own_trails_empty_title => 'Nothing saved yet';

  @override
  String get own_trails_offline_banner =>
      'Offline — showing only trails on this device.';

  @override
  String get trails_on_device => 'Trails (on device)';

  @override
  String get paragraph => '段落';

  @override
  String get pause => 'Pause';

  @override
  String get password => '密码';

  @override
  String get password_confirm => '确认密码';

  @override
  String get passwords_must_match => '两次输入的密码必须一致';

  @override
  String photo_copy_failed_toast(num count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: 'Trail saved, but $count photos couldn\'t be saved.',
      one: 'Trail saved, but 1 photo couldn\'t be saved.',
    );
    return '$_temp0';
  }

  @override
  String get photos => '图片';

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
  String get print => '打印';

  @override
  String get privacy => '隐私';

  @override
  String get private => '非公开的';

  @override
  String get profile => '个人资料';

  @override
  String get public => '公开';

  @override
  String get radius => '半径';

  @override
  String get reached_end_of_trail => 'You\'ve reached the end of the trail.';

  @override
  String get register => '注册';

  @override
  String get required => '必填';

  @override
  String get reorder_photos_hint => 'Long-press and drag to reorder photos.';

  @override
  String get reset => '重置';

  @override
  String get resume => 'Resume';

  @override
  String resume_navigation_prompt(String trail) {
    return 'Resume navigation on $trail?';
  }

  @override
  String get resume_recording_prompt => '恢复录制？';

  @override
  String get road => '道路';

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
  String get follow_roads => 'Follow roads';

  @override
  String get follow_roads_description =>
      'Snap the recorded path to the nearest roads and trails.';

  @override
  String get recalculate_heights => 'Recalculate heights';

  @override
  String get recalculate_heights_description =>
      'Replace recorded GPS elevation with more accurate values from the map.';

  @override
  String get save => '保存';

  @override
  String get save_recording_options => 'Save recording';

  @override
  String get save_track => 'Save track';

  @override
  String get search => 'Search';

  @override
  String get search_for_trails_places => '搜索路线、地点';

  @override
  String get select_date => 'Select date';

  @override
  String get selected => '已选';

  @override
  String get settings => '设置';

  @override
  String get settings_notification_comment_mention => '有人在评论中提到您';

  @override
  String get settings_notification_list_share => '有人与您分享了一个列表';

  @override
  String get settings_notification_new_follower => '您有一位新关注者';

  @override
  String get settings_notification_summit_log_create => '有人在您的轨迹上创建了一个summit日志';

  @override
  String get settings_notification_summit_log_mention => '有人在summit日志中提到了你';

  @override
  String get settings_notification_trail_comment => '有人在你的轨迹上留下了一条评论';

  @override
  String get settings_notification_trail_like => 'Somone liked your trail';

  @override
  String get settings_notification_trail_mention => '有人在轨迹中提到你';

  @override
  String get settings_notification_trail_share => '有人与您分享了一条轨迹';

  @override
  String get settings_privacy_account_private =>
      '只有您可以看到您的个人资料。您将不会出现在搜索结果中。 其他用户不能关注您或与您分享轨迹。您仍然可以发布轨迹或列表。';

  @override
  String get settings_privacy_account_public =>
      '每个人都可以看到您的个人资料。您将出现在搜索结果中。其他用户可以关注您并与您分享轨迹。';

  @override
  String get settings_privacy_lists_private =>
      '您的列表默认是私密的。只有您能够看到它们。 您可以在任何时候修改此设置。';

  @override
  String get settings_privacy_lists_public =>
      '您的列表默认是公开的。每个人都可以看到它们。您可以在任何时候更改此设置为个人列表使用。';

  @override
  String get settings_privacy_trails_private =>
      '您的轨迹默认是私密的。只有您能够看到它们。 您可以在任何时候修改此设置。';

  @override
  String get settings_privacy_trails_public =>
      '您的轨迹默认是公开的。每个人都可以看到它们。您可以在任何时候更改此设置为个人轨迹。';

  @override
  String get share => '分享';

  @override
  String get share_profile => '分享个人资料';

  @override
  String get shared => '共享';

  @override
  String get show_on_map => '地图中展示';

  @override
  String get shower => '淋浴';

  @override
  String signout_unsynced_warning(num count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other:
          'You have $count trails not uploaded yet. Signing out won\'t delete them — they\'ll be here when you sign back in — but they won\'t upload until then.',
      one:
          'You have 1 trail not uploaded yet. Signing out won\'t delete it — it\'ll be here when you sign back in — but it won\'t upload until then.',
    );
    return '$_temp0';
  }

  @override
  String get slogan => '你的路线。你的数据。你的服务器。';

  @override
  String get sort => '排序';

  @override
  String get speed => '速度';

  @override
  String get start => '开始';

  @override
  String get subcategories => '子分类';

  @override
  String get summit_book => '详细日程';

  @override
  String get sync_failed => 'Upload failed · Tap to retry';

  @override
  String get sync_pending => 'Waiting to upload';

  @override
  String get sync_uploading => 'Uploading…';

  @override
  String get table => '表格';

  @override
  String get tags => '标签';

  @override
  String get text => '文本';

  @override
  String get theme_dark => 'Dark';

  @override
  String get theme_light => 'Light';

  @override
  String get theme_system => 'Follow system';

  @override
  String get time => 'Time';

  @override
  String get time_in_motion => 'Time in Motion';

  @override
  String trail(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '路线',
      one: '路线',
    );
    return '$_temp0';
  }

  @override
  String get trail_saved_successfully => '路线保存成功';

  @override
  String get trail_not_on_this_device =>
      'This trail is no longer on this device.';

  @override
  String get trail_uploaded_reopen_to_edit =>
      'This trail finished uploading. Re-open it from your trails to keep editing.';

  @override
  String get some_waypoints_failed_to_save =>
      'Trail saved, but some waypoints failed to save';

  @override
  String get units => '单位';

  @override
  String get users => 'Users';

  @override
  String get username => '用户名';

  @override
  String get username_not_unique => '此用户名已被使用。请尝试其他用户名。';

  @override
  String get view => '查看';

  @override
  String get visibilty_status => '可见状态';

  @override
  String get water => '水';

  @override
  String get web => 'Web';

  @override
  String waypoints(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '路点',
      one: '路点',
    );
    return '$_temp0';
  }

  @override
  String get welcome_to => 'Welcome to';

  @override
  String get width => '宽度';

  @override
  String get wrong_username_or_password => '用户名或密码无效';

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

  @override
  String get something_went_wrong => 'Something went wrong';

  @override
  String get technical_details => 'Technical Details';

  @override
  String get link => 'Link';

  @override
  String get url => 'URL';

  @override
  String get open_in_new_tab => '在新标签页中打开';

  @override
  String get remove => 'Remove';

  @override
  String get remove_download_confirm_body =>
      'This removes the downloaded copy from this device. The trail itself is not deleted — you\'ll need to download it again to use it offline.';

  @override
  String get apply => 'Apply';

  @override
  String get add_at_least_2_anchors_hint =>
      'Add at least 2 anchors to see the elevation profile.';

  @override
  String get reverse_direction => '反向';

  @override
  String get delete_all => 'Delete all';

  @override
  String get auto_routing => 'Auto-routing';

  @override
  String get auto_routing_hint =>
      'Automatically follow roads and paths between anchors.';

  @override
  String get travel_profile => 'Travel profile';

  @override
  String get no_track_data => 'No track data';

  @override
  String get offline => 'Offline';

  @override
  String get available_offline => 'Available offline';

  @override
  String get no_lists_found => 'No lists found';

  @override
  String get search_lists => 'Search lists…';

  @override
  String get search_for_a_location => 'Search for a location';

  @override
  String no_results_for_query(String query) {
    return 'No results for \"$query\"';
  }

  @override
  String get filter => 'Filter';

  @override
  String no_label_yet(String label) {
    return 'No $label yet.';
  }

  @override
  String get no_lists_yet => 'No lists yet.';

  @override
  String get no_bio_yet => 'No bio yet.';

  @override
  String get feed => 'Feed';

  @override
  String get no_trails_yet => 'No trails yet.';

  @override
  String get library_empty_title => 'No downloaded trails';

  @override
  String get library_empty_body =>
      'Trails you download are kept here so you can open them offline.';

  @override
  String get library_empty_search_body =>
      'Try a different search term or clear your filters.';

  @override
  String get search_location => 'Search location';

  @override
  String no_servers_match_query(String query) {
    return 'No servers match \"$query\"';
  }

  @override
  String get use_custom_url_instead => 'Use custom URL instead';

  @override
  String get select_instance => 'Select Instance';

  @override
  String get enter_server_url_hint => 'Enter server URL (e.g. wanderer.to)';

  @override
  String get search_library => 'Search library…';

  @override
  String language_and_units(String language, String units) {
    return '$language & $units';
  }

  @override
  String get edit_route => '编辑路线';

  @override
  String get undo => 'Undo';

  @override
  String get redo => 'Redo';

  @override
  String get bold => 'Bold';

  @override
  String get italic => 'Italic';

  @override
  String get underline => 'Underline';

  @override
  String get bullet_list => 'Bullet list';

  @override
  String get ordered_list => 'Ordered list';

  @override
  String get blockquote => 'Blockquote';

  @override
  String get library => 'Library';

  @override
  String get settings_offline_regions_title => 'Offline Maps/Regions';

  @override
  String get regions_search_hint => 'Search regions';

  @override
  String get regions_dem_toggle_label => 'Download elevation data (DEM)';

  @override
  String get regions_dem_toggle_caption =>
      'Adds hillshading; increases download size';

  @override
  String get regions_update_available => 'Update available';

  @override
  String get regions_update_action => 'Update';

  @override
  String get regions_retry => 'Retry';

  @override
  String get regions_not_yet_available => 'Not yet available';

  @override
  String get regions_build_failed => 'Build failed';

  @override
  String regions_delete_confirm_title(String name) {
    return 'Delete $name?';
  }

  @override
  String get regions_delete_confirm_body =>
      'This removes the downloaded map and elevation data for this region. You\'ll need to download it again to use it offline.';

  @override
  String get regions_delete_confirm_action => 'Delete';

  @override
  String regions_disk_usage_summary(String size, num count) {
    return '$size used across $count downloaded region(s)';
  }

  @override
  String get regions_empty_search_title => 'No matching regions';

  @override
  String get regions_empty_search_body => 'Try a different search term.';

  @override
  String get regions_empty_catalog_title => 'No offline regions available';

  @override
  String get regions_empty_catalog_body =>
      'Ask your Wanderer instance administrator to configure downloadable regions.';

  @override
  String get regions_vector_tile_title => 'Vector';

  @override
  String get regions_dem_tile_title => 'Elevation data';

  @override
  String get regions_download_failed => 'Download failed';

  @override
  String get regions_dem_locked_subtitle => 'Download map data first';

  @override
  String get regions_offline_unavailable_title => 'Can\'t load regions';

  @override
  String get regions_offline_unavailable_body =>
      'Connect to the internet to browse and manage downloadable regions.';

  @override
  String regions_group_expand_label(String name) {
    return 'Expand $name';
  }

  @override
  String regions_group_collapse_label(String name) {
    return 'Collapse $name';
  }

  @override
  String get offline_title => 'You\'re offline';

  @override
  String get offline_try_again => 'Try again';

  @override
  String get offline_map_body =>
      'Connect to the internet to load the map. Downloaded trails are still available.';

  @override
  String get offline_list_body => 'Connect to the internet to load lists.';

  @override
  String get offline_profile_body =>
      'Connect to the internet to load your full profile.';

  @override
  String get offline_settings_banner =>
      'You\'re offline. Settings are read-only until you reconnect.';

  @override
  String get offline_action_unavailable =>
      'You\'re offline — try again once you\'re back online.';

  @override
  String get offline_categories_body =>
      'Connect to the internet to manage categories.';

  @override
  String get offline_trail_search_body =>
      'Connect to the internet to search for trails. Downloaded trails are still available.';
}
